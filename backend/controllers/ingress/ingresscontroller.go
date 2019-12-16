package ingress

import (
	"fmt"
	"time"

	"kubecloud/backend/controllers/util"
	dao "kubecloud/backend/dao"

	"github.com/astaxie/beego"

	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	extensionsinformers "k8s.io/client-go/informers/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	extensionslisters "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	// maxRetries is the number of times a ingress will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a ingress is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// IngressController is responsible for synchronizing ingress objects stored
type IngressController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncIngress for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueIngress func(ing *extensions.Ingress)

	// ingLister can list/get ingresses from the shared informer's store
	ingLister extensionslisters.IngressLister

	// ingListerSynced returns true if the ingress store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	ingListerSynced cache.InformerSynced

	// Ingresses that need to be synced
	queue workqueue.RateLimitingInterface
	// ingress dbhandler
	kubeIngHandler *dao.K8sIngressModel
}

// NewIngressController creates a new IngressController.
func NewIngressController(cluster string,
	client kubernetes.Interface,
	ingInformer extensionsinformers.IngressInformer) (*IngressController, error) {
	ic := &IngressController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress"),
	}

	ingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ic.addIngress,
		UpdateFunc: ic.updateIngress,
		DeleteFunc: ic.deleteIngress,
	})
	ic.syncHandler = ic.syncIngress
	ic.enqueueIngress = ic.enqueue

	ic.ingLister = ingInformer.Lister()
	ic.ingListerSynced = ingInformer.Informer().HasSynced

	ic.kubeIngHandler = dao.NewK8sIngressModel()

	return ic, nil
}

// Run begins watching and syncing.
func (ic *IngressController) Run(workers int, stopCh <-chan struct{}) {
	defer ic.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, ic.ingListerSynced) {
		beego.Error("ingress controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(ic.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (ic *IngressController) addIngress(obj interface{}) {
	ing := obj.(*extensions.Ingress)
	ic.enqueueIngress(ing)
}

func (ic *IngressController) updateIngress(old, cur interface{}) {
	//olding := old.(*extensions.Ingress)
	curing := cur.(*extensions.Ingress)
	ic.enqueueIngress(curing)
}

func (ic *IngressController) deleteIngress(obj interface{}) {
	ing, ok := obj.(*extensions.Ingress)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		ing, ok = tombstone.Obj.(*extensions.Ingress)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a Ingress %#v", obj))
			return
		}
	}
	ic.enqueueIngress(ing)
}

func (ic *IngressController) enqueue(ing *extensions.Ingress) {
	key, err := controller.KeyFunc(ing)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", ing, err))
		return
	}

	ic.queue.Add(key)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (ic *IngressController) worker() {
	for ic.processNextWorkItem() {
	}
}

func (ic *IngressController) processNextWorkItem() bool {
	key, quit := ic.queue.Get()
	if quit {
		beego.Error("get item from workqueue failed!")
		return false
	}
	defer ic.queue.Done(key)

	err := ic.syncHandler(key.(string))
	ic.handleErr(err, key)

	return true
}

func (ic *IngressController) handleErr(err error, key interface{}) {
	if err == nil {
		ic.queue.Forget(key)
		return
	}

	if ic.queue.NumRequeues(key) < maxRetries {
		ic.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping ingress %q out of the queue: %v, cluster: %s", key, err, ic.cluster))
	ic.queue.Forget(key)
}

// syncIngress will sync the ingress with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (ic *IngressController) syncIngress(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	//check namespace
	if util.FilterNamespace(ic.cluster, namespace) {
		//beego.Warn("Skip this syncing of cluster "+ic.cluster, namespace)
		return nil
	}
	ing, err := ic.ingLister.Ingresses(namespace).Get(name)
	if errors.IsNotFound(err) {
		err = ic.deleteIngressRecord(namespace, name)
		if err != nil {
			beego.Error("Delete ingress from database failed: ", err)
		}
		return err
	}
	if err != nil {
		return err
	}

	return ic.syncIngressRecord(*ing)
}
