package application

import (
	"fmt"
	"time"

	"kubecloud/backend/controllers/util"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"

	v1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/apps/v1beta1"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	// maxRetries is the number of times a deployment will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a deployment is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1beta1.SchemeGroupVersion.WithKind("Deployment")

// DeploymentController is responsible for synchronizing Deployment objects stored
// in the system with actual running replica sets and pods.
type ApplicationController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncDeployment for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueDeployment func(deployment *v1beta1.Deployment)

	// dLister can list/get deployments from the shared informer's store
	dLister listers.DeploymentLister

	// dListerSynced returns true if the Deployment store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	dListerSynced cache.InformerSynced

	// Deployments that need to be synced
	queue workqueue.RateLimitingInterface
	// application dbhandler
	syncAppHandler *syncApplication
}

// NewDeploymentController creates a new DeploymentController.
func NewApplicationController(cluster string,
	client kubernetes.Interface,
	dInformer informers.DeploymentInformer) (*ApplicationController, error) {
	ac := &ApplicationController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "deployment"),
	}

	dInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ac.addDeployment,
		UpdateFunc: ac.updateDeployment,
		DeleteFunc: ac.deleteDeployment,
	})
	ac.syncHandler = ac.syncDeployment
	ac.enqueueDeployment = ac.enqueue

	ac.dLister = dInformer.Lister()
	ac.dListerSynced = dInformer.Informer().HasSynced

	ac.syncAppHandler = newSyncApplication(cluster)

	return ac, nil
}

// Run begins watching and syncing.
func (ac *ApplicationController) Run(workers int, stopCh <-chan struct{}) {
	defer ac.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, ac.dListerSynced) {
		beego.Error("application controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(ac.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (ac *ApplicationController) addDeployment(obj interface{}) {
	d := obj.(*v1beta1.Deployment)
	ac.enqueueDeployment(d)
}

func (ac *ApplicationController) updateDeployment(old, cur interface{}) {
	//oldD := old.(*v1beta1.Deployment)
	curD := cur.(*v1beta1.Deployment)
	ac.enqueueDeployment(curD)
}

func (ac *ApplicationController) deleteDeployment(obj interface{}) {
	d, ok := obj.(*v1beta1.Deployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*v1beta1.Deployment)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a Deployment %#v", obj))
			return
		}
	}
	ac.enqueueDeployment(d)
}

func (ac *ApplicationController) enqueue(deployment *v1beta1.Deployment) {
	key, err := controller.KeyFunc(deployment)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", deployment, err))
		return
	}

	ac.queue.Add(key)
}

func (ac *ApplicationController) enqueueRateLimited(deployment *v1beta1.Deployment) {
	key, err := controller.KeyFunc(deployment)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", deployment, err))
		return
	}

	ac.queue.AddRateLimited(key)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (ac *ApplicationController) worker() {
	for ac.processNextWorkItem() {
	}
}

func (ac *ApplicationController) processNextWorkItem() bool {
	key, quit := ac.queue.Get()
	if quit {
		beego.Debug("get item from workqueue failed!")
		return false
	}
	defer ac.queue.Done(key)

	err := ac.syncHandler(key.(string))
	ac.handleErr(err, key)

	return true
}

func (ac *ApplicationController) handleErr(err error, key interface{}) {
	if err == nil {
		ac.queue.Forget(key)
		return
	}

	if ac.queue.NumRequeues(key) < maxRetries {
		ac.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping deployment %q out of the queue: %v, cluster: %s", key, err, ac.cluster))
	ac.queue.Forget(key)
}

// syncDeployment will sync the deployment with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (ac *ApplicationController) syncDeployment(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	//check namespace
	if util.FilterNamespace(ac.cluster, namespace) {
		//beego.Warn("Skip this syncing of cluster "+ac.cluster, namespace)
		return nil
	}
	deployment, err := ac.dLister.Deployments(namespace).Get(name)
	if errors.IsNotFound(err) {
		beego.Info(fmt.Sprintf("Deployment %v has been deleted, cluster: %s", key, ac.cluster))
		_, err := ac.syncAppHandler.appHandler.GetAppByName(ac.cluster, namespace, name)
		if err != nil {
			if err == orm.ErrNoRows {
				return nil
			}
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	return ac.syncAppHandler.syncDeployApplication(*deployment)
}
