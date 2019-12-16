package namespace

import (
	"fmt"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	// maxRetries is the number of times a namespace will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a namespace is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

// NamespaceController is responsible for synchronizing namespace objects stored
// in the system with actual running replica sets and pods.
type NamespaceController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncNamespace for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueNamespace func(obj *core.Namespace)

	namespaceLister corelisters.NamespaceLister

	namespaceListerSynced cache.InformerSynced

	// Namespaces that need to be synced
	queue workqueue.RateLimitingInterface
	// Namespace model
	namespaceModel *dao.NamespaceModel
}

// NewNamespaceController creates a new NamespaceController.
func NewNamespaceController(cluster string,
	client kubernetes.Interface,
	sInformer coreinformers.NamespaceInformer) *NamespaceController {
	nc := &NamespaceController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "namespace"),
	}
	sInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nc.addNamespace,
		UpdateFunc: nc.updateNamespace,
		DeleteFunc: nc.deleteNamespace,
	})
	nc.syncHandler = nc.syncNamespaceFromKey
	nc.enqueueNamespace = nc.enqueue
	nc.namespaceLister = sInformer.Lister()
	nc.namespaceListerSynced = sInformer.Informer().HasSynced
	nc.namespaceModel = dao.NewNamespaceModel()

	return nc
}

// Run begins watching and syncing.
func (nc *NamespaceController) Run(workers int, stopCh <-chan struct{}) {
	defer nc.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, nc.namespaceListerSynced) {
		beego.Error("namespace controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(nc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (nc *NamespaceController) enqueue(obj *core.Namespace) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", obj, err))
		return
	}

	nc.queue.Add(key)
}

func (nc *NamespaceController) addNamespace(obj interface{}) {
	s := obj.(*core.Namespace)
	if s.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		nc.deleteNamespace(s)
		return
	}

	s, err := nc.namespaceLister.Get(s.Name)
	if err != nil {
		beego.Error("namespace added sync failed for:", err)
		return
	}
	nc.enqueueNamespace(s)
}

func (nc *NamespaceController) updateNamespace(old, cur interface{}) {
	curobj := cur.(*core.Namespace)
	oldobj := old.(*core.Namespace)
	if curobj.ResourceVersion == oldobj.ResourceVersion {
		// Periodic resync will send update events for all known replica sets.
		// Two different versions of the same replica set will always have different RVs.
		return
	}
	s, err := nc.namespaceLister.Get(curobj.Name)
	if err != nil {
		beego.Error("namespace update sync failed for:", err)
		return
	}
	nc.enqueueNamespace(s)
}

func (nc *NamespaceController) deleteNamespace(obj interface{}) {
	s, ok := obj.(*core.Namespace)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		s, ok = tombstone.Obj.(*core.Namespace)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a namespace %#v", obj))
			return
		}
	}
	nc.enqueueNamespace(s)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (nc *NamespaceController) worker() {
	for nc.processNextWorkItem() {
	}
}

func (nc *NamespaceController) processNextWorkItem() bool {
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

func (nc *NamespaceController) handleErr(err error, key interface{}) {
	if err == nil {
		nc.queue.Forget(key)
		return
	}

	if nc.queue.NumRequeues(key) < maxRetries {
		nc.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping namespace %q out of the queue: %v, cluster: %s", key, err, nc.cluster))
	nc.queue.Forget(key)
}

// syncNamespaceFromKey looks for a namespace with the specified key in its store and synchronizes it
func (nc *NamespaceController) syncNamespaceFromKey(key string) error {
	// startTime := time.Now()
	// defer func() {
	// 	beego.Info("NamespaceController finished syncing namespace", key, time.Now().Sub(startTime))
	// }()

	namespace, err := nc.namespaceLister.Get(key)
	if errors.IsNotFound(err) {
		// namespace has been deleted
		if err := dao.NamespaceDelete(nc.cluster, key); err != nil {
			beego.Error("NamespaceController error:", err.Error())
		}
		return nil
	}
	if err != nil {
		beego.Error("NamespaceController error:", err.Error())
		return err
	}
	if namespace == nil {
		return nil
	}
	if namespace.Status.Phase == core.NamespaceTerminating {
		// namespace is terminating
		if err := dao.NamespaceDelete(nc.cluster, key); err != nil && err != orm.ErrNoRows {
			beego.Error("NamespaceController error:", err.Error())
		}
		return nil
	}
	// add namespace to database
	if row, err := dao.NamespaceGet(nc.cluster, key); err != nil {
		if err != orm.ErrNoRows {
			beego.Error("NamespaceController error:", err.Error())
			return err
		}
		row = &models.K8sNamespace{
			Cluster:     nc.cluster,
			Name:        key,
			CPUQuota:    "",
			MemoryQuota: "",
			AddonsUnix:  models.NewAddonsUnix(),
		}
		row.CreatedAt = namespace.CreationTimestamp.Unix()
		// fetch quota from k8s
		if quotaList, err := nc.client.CoreV1().ResourceQuotas(key).List(metav1.ListOptions{}); err == nil {
			quotaName := resource.GenResourceQuotaName(key)
			for _, item := range quotaList.Items {
				if item.Name == quotaName {
					if quantity, ok := item.Spec.Hard[core.ResourceLimitsCPU]; ok {
						row.CPUQuota = quantity.String()
					}
					if quantity, ok := item.Spec.Hard[core.ResourceLimitsMemory]; ok {
						row.MemoryQuota = quantity.String()
					}
					break
				}
			}
		}
		if err := dao.NamespaceInsert(row); err != nil {
			beego.Error("NamespaceController error:", err.Error())
		}
	}
	return nil
}
