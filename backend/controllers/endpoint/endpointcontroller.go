package endpoint

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
)

const (
	// maxRetries is the number of times a endpoint will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a endpoint is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// EndpointController is responsible for synchronizing endpoint objects stored
// in the system with actual running replica sets and pods.
type EndpointController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncEndpoint for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueEndpoint func(endpoint *core.Endpoints)

	endpointLister corelisters.EndpointsLister

	endpointListerSynced cache.InformerSynced

	// Endpoints that need to be synced
	queue workqueue.RateLimitingInterface
	// endpoint dbhandler
	endpointHandler *dao.K8sEndpointModel
}

// NewEndpointController creates a new EndpointController.
func NewEndpointController(cluster string,
	client kubernetes.Interface,
	endpointInformer coreinformers.EndpointsInformer) (*EndpointController, error) {
	ec := &EndpointController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "endpoint"),
	}
	endpointInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ec.addEndpoint,
		UpdateFunc: ec.updateEndpoint,
		DeleteFunc: ec.deleteEndpoint,
	})
	ec.syncHandler = ec.syncEndpoint
	ec.enqueueEndpoint = ec.enqueue
	ec.endpointLister = endpointInformer.Lister()
	ec.endpointListerSynced = endpointInformer.Informer().HasSynced
	ec.endpointHandler = dao.NewK8sEndpointModel()

	return ec, nil
}

// Run begins watching and syncing.
func (ec *EndpointController) Run(workers int, stopCh <-chan struct{}) {
	defer ec.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, ec.endpointListerSynced) {
		beego.Error("endpoint controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(ec.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (ec *EndpointController) enqueue(endpoint *core.Endpoints) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(endpoint)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", endpoint, err))
		return
	}

	ec.queue.Add(key)
}

func (ec *EndpointController) addEndpoint(obj interface{}) {
	ep := obj.(*core.Endpoints)
	if ep.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		ec.deleteEndpoint(ep)
		return
	}

	endpoint, err := ec.endpointLister.Endpoints(ep.Namespace).Get(ep.Name)
	if err != nil {
		beego.Error("Endpoint added sync failed for:", err)
		return
	}
	ec.enqueueEndpoint(endpoint)
}

func (ec *EndpointController) updateEndpoint(old, cur interface{}) {
	curep := cur.(*core.Endpoints)
	oldep := old.(*core.Endpoints)
	if curep.ResourceVersion == oldep.ResourceVersion {
		// Periodic resync will send update events for all known replica sets.
		// Two different versions of the same replica set will always have different RVs.
		return
	}
	s, err := ec.endpointLister.Endpoints(curep.Namespace).Get(curep.Name)
	if err != nil {
		beego.Error("Endpoint update sync failed for:", err)
		return
	}
	ec.enqueueEndpoint(s)
}

func (ec *EndpointController) deleteEndpoint(obj interface{}) {
	ep, ok := obj.(*core.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		ep, ok = tombstone.Obj.(*core.Endpoints)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a endpoint %#v", obj))
			return
		}
	}
	ec.enqueueEndpoint(ep)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (ec *EndpointController) worker() {
	for ec.processNextWorkItem() {
	}
}

func (ec *EndpointController) processNextWorkItem() bool {
	key, quit := ec.queue.Get()
	if quit {
		beego.Debug("Get item from workqueue failed!")
		return false
	}
	defer ec.queue.Done(key)

	err := ec.syncHandler(key.(string))
	ec.handleErr(err, key)

	return true
}

func (ec *EndpointController) handleErr(err error, key interface{}) {
	if err == nil {
		ec.queue.Forget(key)
		return
	}

	if ec.queue.NumRequeues(key) < maxRetries {
		ec.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping endpoint %q out of the queue: %v, cluster: %s", key, err, ec.cluster))
	ec.queue.Forget(key)
}

// syncEndpoint will sync the endpoint with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (ec *EndpointController) syncEndpoint(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	//check namespace
	if util.FilterNamespace(ec.cluster, namespace) {
		return nil
	}
	svc, err := ec.endpointLister.Endpoints(namespace).Get(name)
	if errors.IsNotFound(err) {
		beego.Debug(fmt.Sprintf("Endpoint %v has been deleted, cluster: %s", key, ec.cluster))
		err = ec.deleteEndpointRecord(namespace, name) //todo delete it from database
		if err != nil {
			beego.Error("Delete endpoint from database failed: ", err)
		}
		return err
	}
	if err != nil {
		return err
	}

	return ec.syncEndpointRecord(*svc)
}
