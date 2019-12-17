package resourcequota

import (
	"fmt"
	"time"

	"kubecloud/backend/dao"

	"github.com/astaxie/beego"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// maxRetries is the number of times a resourcequota will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a resourcequota is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// ResourceQuotaController is responsible for synchronizing resourcequota objects stored
// in the system with actual running replica sets and pods.
type ResourceQuotaController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncResourceQuota for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueResourceQuota func(obj *core.ResourceQuota)

	resourcequotaLister corelisters.ResourceQuotaLister

	resourcequotaListerSynced cache.InformerSynced

	// ResourceQuotas that need to be synced
	queue workqueue.RateLimitingInterface
	// ResourceQuota model
	namespaceModel *dao.NamespaceModel
}

// NewResourceQuotaController creates a new ResourceQuotaController.
func NewResourceQuotaController(cluster string,
	client kubernetes.Interface,
	sInformer coreinformers.ResourceQuotaInformer) *ResourceQuotaController {
	nc := &ResourceQuotaController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "resourcequota"),
	}
	sInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nc.addResourceQuota,
		UpdateFunc: nc.updateResourceQuota,
		DeleteFunc: nc.deleteResourceQuota,
	})
	nc.syncHandler = nc.syncResourceQuotaFromKey
	nc.enqueueResourceQuota = nc.enqueue
	nc.resourcequotaLister = sInformer.Lister()
	nc.resourcequotaListerSynced = sInformer.Informer().HasSynced
	nc.namespaceModel = dao.NewNamespaceModel()

	return nc
}

// Run begins watching and syncing.
func (nc *ResourceQuotaController) Run(workers int, stopCh <-chan struct{}) {
	defer nc.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, nc.resourcequotaListerSynced) {
		beego.Error("resourcequota controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(nc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (nc *ResourceQuotaController) enqueue(obj *core.ResourceQuota) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", obj, err))
		return
	}

	nc.queue.Add(key)
}

func (nc *ResourceQuotaController) addResourceQuota(obj interface{}) {
	cur := obj.(*core.ResourceQuota)
	if cur.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		nc.deleteResourceQuota(cur)
		return
	}

	item, err := nc.resourcequotaLister.ResourceQuotas(cur.Namespace).Get(cur.Name)
	if err != nil {
		beego.Error("resourcequota added sync failed for:", err)
		return
	}
	nc.enqueueResourceQuota(item)
}

func (nc *ResourceQuotaController) updateResourceQuota(oldobj, curobj interface{}) {
	cur := curobj.(*core.ResourceQuota)
	old := oldobj.(*core.ResourceQuota)
	if cur.ResourceVersion == old.ResourceVersion {
		// Periodic resync will send update events for all known replica sets.
		// Two different versions of the same replica set will always have different RVs.
		return
	}
	item, err := nc.resourcequotaLister.ResourceQuotas(cur.Namespace).Get(cur.Name)
	if err != nil {
		beego.Error("resourcequota update sync failed for:", err)
		return
	}
	nc.enqueueResourceQuota(item)
}

func (nc *ResourceQuotaController) deleteResourceQuota(obj interface{}) {
	s, ok := obj.(*core.ResourceQuota)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		s, ok = tombstone.Obj.(*core.ResourceQuota)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a resourcequota %#v", obj))
			return
		}
	}
	nc.enqueueResourceQuota(s)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (nc *ResourceQuotaController) worker() {
	for nc.processNextWorkItem() {
	}
}

func (nc *ResourceQuotaController) processNextWorkItem() bool {
	key, quit := nc.queue.Get()
	if quit {
		beego.Debug("get item from workqueue failed!")
		return false
	}
	defer nc.queue.Done(key)

	err := nc.syncHandler(key.(string))
	nc.handleErr(err, key)

	return true
}

func (nc *ResourceQuotaController) handleErr(err error, key interface{}) {
	if err == nil {
		nc.queue.Forget(key)
		return
	}

	if nc.queue.NumRequeues(key) < maxRetries {
		nc.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping resourcequota %q out of the queue: %v, cluster: %s", key, err, nc.cluster))
	nc.queue.Forget(key)
}

// syncResourceQuotaFromKey looks for a resourcequota with the specified key in its store and synchronizes it
func (nc *ResourceQuotaController) syncResourceQuotaFromKey(key string) error {
	// startTime := time.Now()
	// defer func() {
	// 	beego.Info("ResourceQuotaController finished syncing resourcequota", key, time.Now().Sub(startTime))
	// }()
	//
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	quota, err := nc.resourcequotaLister.ResourceQuotas(namespace).Get(name)
	if errors.IsNotFound(err) {
		// don't do anything
		return nil
	}
	if err != nil {
		beego.Error("ResourceQuotaController error:", err.Error())
		return err
	}
	// update quota in database
	findMinQuantity := func(list core.ResourceList, names []core.ResourceName) (quantity *k8sresource.Quantity) {
		for _, name := range names {
			if val, ok := list[name]; ok {
				if quantity == nil || val.Cmp(*quantity) < 0 {
					quantity = &val
				}
			}
		}
		return quantity
	}

	row, err := dao.NamespaceGet(nc.cluster, namespace)
	if err != nil {
		beego.Error("ResourceQuotaController error:", err.Error())
		return err
	}
	cpuNames := []core.ResourceName{core.ResourceCPU, core.ResourceLimitsCPU, core.ResourceRequestsCPU}
	if quantity := findMinQuantity(quota.Spec.Hard, cpuNames); quantity != nil {
		row.CPUQuota = quantity.String()
	} else {
		row.CPUQuota = ""
	}
	memoryNames := []core.ResourceName{core.ResourceMemory, core.ResourceLimitsMemory, core.ResourceRequestsMemory}
	if quantity := findMinQuantity(quota.Spec.Hard, memoryNames); quantity != nil {
		row.MemoryQuota = quantity.String()
	} else {
		row.MemoryQuota = ""
	}
	row.MarkUpdated()
	if err := dao.NamespaceUpdate(row); err != nil {
		beego.Error("ResourceQuotaController error:", err.Error())
		return err
	}

	return nil
}
