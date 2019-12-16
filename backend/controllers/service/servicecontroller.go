package service

import (
	"fmt"
	"time"

	"kubecloud/backend/controllers/util"
	dao "kubecloud/backend/dao"

	"github.com/astaxie/beego"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	// maxRetries is the number of times a service will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a service is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// ServiceController is responsible for synchronizing service objects stored
// in the system with actual running replica sets and pods.
type ServiceController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncService for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueService func(svc *core.Service)

	svcLister corelisters.ServiceLister

	svcListerSynced cache.InformerSynced

	// Services that need to be synced
	queue workqueue.RateLimitingInterface
	// service dbhandler
	svcHandler *dao.K8sServiceModel
}

// NewServiceController creates a new ServiceController.
func NewServiceController(cluster string,
	client kubernetes.Interface,
	svcInformer coreinformers.ServiceInformer) (*ServiceController, error) {
	sc := &ServiceController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "service"),
	}
	svcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    sc.addService,
		UpdateFunc: sc.updateService,
		DeleteFunc: sc.deleteService,
	})
	sc.syncHandler = sc.syncService
	sc.enqueueService = sc.enqueue
	sc.svcLister = svcInformer.Lister()
	sc.svcListerSynced = svcInformer.Informer().HasSynced
	sc.svcHandler = dao.NewK8sServiceModel()

	return sc, nil
}

// Run begins watching and syncing.
func (sc *ServiceController) Run(workers int, stopCh <-chan struct{}) {
	defer sc.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, sc.svcListerSynced) {
		beego.Error("service controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(sc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (sc *ServiceController) enqueue(svc *core.Service) {
	key, err := controller.KeyFunc(svc)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", svc, err))
		return
	}

	sc.queue.Add(key)
}

func (sc *ServiceController) addService(obj interface{}) {
	svc := obj.(*core.Service)
	if svc.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		sc.deleteService(svc)
		return
	}

	s, err := sc.svcLister.Services(svc.Namespace).Get(svc.Name)
	if err != nil {
		beego.Error("service added sync failed for:", err)
		return
	}
	sc.enqueueService(s)
}

func (sc *ServiceController) updateService(old, cur interface{}) {
	curSvc := cur.(*core.Service)
	oldSvc := old.(*core.Service)
	if curSvc.ResourceVersion == oldSvc.ResourceVersion {
		// Periodic resync will send update events for all known replica sets.
		// Two different versions of the same replica set will always have different RVs.
		return
	}
	s, err := sc.svcLister.Services(curSvc.Namespace).Get(curSvc.Name)
	if err != nil {
		beego.Error("service update sync failed for:", err)
		return
	}
	sc.enqueueService(s)
}

func (sc *ServiceController) deleteService(obj interface{}) {
	svc, ok := obj.(*core.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		svc, ok = tombstone.Obj.(*core.Service)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a service %#v", obj))
			return
		}
	}
	sc.enqueueService(svc)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (sc *ServiceController) worker() {
	for sc.processNextWorkItem() {
	}
}

func (sc *ServiceController) processNextWorkItem() bool {
	key, quit := sc.queue.Get()
	if quit {
		beego.Debug("get item from workqueue failed!")
		return false
	}
	defer sc.queue.Done(key)

	err := sc.syncHandler(key.(string))
	sc.handleErr(err, key)

	return true
}

func (sc *ServiceController) handleErr(err error, key interface{}) {
	if err == nil {
		sc.queue.Forget(key)
		return
	}

	if sc.queue.NumRequeues(key) < maxRetries {
		sc.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping service %q out of the queue: %v, cluster: %s", key, err, sc.cluster))
	sc.queue.Forget(key)
}

// syncService will sync the service with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (sc *ServiceController) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	//check namespace
	if util.FilterNamespace(sc.cluster, namespace) {
		//beego.Warn("Skip this syncing of cluster "+sc.cluster, namespace)
		return nil
	}
	svc, err := sc.svcLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		beego.Debug(fmt.Sprintf("Service %v has been deleted, cluster: %s", key, sc.cluster))
		err = sc.deleteServiceRecord(namespace, name) //todo delete it from database
		if err != nil {
			beego.Error("delete service from database failed: ", err)
		}
		return err
	}
	if err != nil {
		return err
	}

	return sc.syncServiceRecord(*svc)
}
