package secret

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
	// maxRetries is the number of times a secret will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a secret is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 10
)

var syncSecretType = []core.SecretType{core.SecretTypeTLS, core.SecretTypeBasicAuth, core.SecretTypeOpaque}

// SecretController is responsible for synchronizing secret objects stored
// in the system with actual running replica sets and pods.
type SecretController struct {
	cluster string
	client  kubernetes.Interface

	// To allow injection of syncSecret for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueSecret func(obj *core.Secret)

	secretLister corelisters.SecretLister

	secretListerSynced cache.InformerSynced

	// Secrets that need to be synced
	queue workqueue.RateLimitingInterface
	// secret dbhandler
	secretHandler *dao.SecretModel
}

// NewSecretController creates a new SecretController.
func NewSecretController(cluster string,
	client kubernetes.Interface,
	sInformer coreinformers.SecretInformer) (*SecretController, error) {
	sc := &SecretController{
		cluster: cluster,
		client:  client,
		queue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "secret"),
	}
	sInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    sc.addSecret,
		UpdateFunc: sc.updateSecret,
		DeleteFunc: sc.deleteSecret,
	})
	sc.syncHandler = sc.syncSecret
	sc.enqueueSecret = sc.enqueue
	sc.secretLister = sInformer.Lister()
	sc.secretListerSynced = sInformer.Informer().HasSynced
	sc.secretHandler = dao.NewSecretModel()

	return sc, nil
}

// Run begins watching and syncing.
func (sc *SecretController) Run(workers int, stopCh <-chan struct{}) {
	defer sc.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, sc.secretListerSynced) {
		beego.Error("secret controller cache sync failed!")
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(sc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (sc *SecretController) enqueue(obj *core.Secret) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", obj, err))
		return
	}

	sc.queue.Add(key)
}

func (sc *SecretController) addSecret(obj interface{}) {
	s := obj.(*core.Secret)
	if s.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		sc.deleteSecret(s)
		return
	}

	s, err := sc.secretLister.Secrets(s.Namespace).Get(s.Name)
	if err != nil {
		beego.Error("secret added sync failed for:", err)
		return
	}
	sc.enqueueSecret(s)
}

func (sc *SecretController) updateSecret(old, cur interface{}) {
	curobj := cur.(*core.Secret)
	oldobj := old.(*core.Secret)
	if curobj.ResourceVersion == oldobj.ResourceVersion {
		// Periodic resync will send update events for all known replica sets.
		// Two different versions of the same replica set will always have different RVs.
		return
	}
	s, err := sc.secretLister.Secrets(curobj.Namespace).Get(curobj.Name)
	if err != nil {
		beego.Error("secret update sync failed for:", err)
		return
	}
	sc.enqueueSecret(s)
}

func (sc *SecretController) deleteSecret(obj interface{}) {
	s, ok := obj.(*core.Secret)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			beego.Error(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		s, ok = tombstone.Obj.(*core.Secret)
		if !ok {
			beego.Error(fmt.Errorf("Tombstone contained object that is not a secret %#v", obj))
			return
		}
	}
	sc.enqueueSecret(s)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (sc *SecretController) worker() {
	for sc.processNextWorkItem() {
	}
}

func (sc *SecretController) processNextWorkItem() bool {
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

func (sc *SecretController) handleErr(err error, key interface{}) {
	if err == nil {
		sc.queue.Forget(key)
		return
	}

	if sc.queue.NumRequeues(key) < maxRetries {
		sc.queue.AddRateLimited(key)
		return
	}

	beego.Warn(fmt.Sprintf("Dropping secret %q out of the queue: %v, cluster: %s", key, err, sc.cluster))
	sc.queue.Forget(key)
}

// syncSecret will sync the secret with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (sc *SecretController) syncSecret(key string) error {
	filterSecretType := func(target core.SecretType) bool {
		for _, item := range syncSecretType {
			if target == item {
				return false
			}
		}
		return true
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	//check namespace
	if util.FilterNamespace(sc.cluster, namespace) {
		//beego.Warn("Skip this syncing of cluster "+sc.cluster, namespace)
		return nil
	}
	s, err := sc.secretLister.Secrets(namespace).Get(name)
	if err == nil {
		if filterSecretType(s.Type) {
			//beego.Warn("Skip this syncing of cluster "+sc.cluster, namespace, string(s.Type)+" is filtered!")
			return nil
		}
	} else {
		if errors.IsNotFound(err) {
			beego.Debug(fmt.Sprintf("Secret %v has been deleted, cluster: %s", key, sc.cluster))
			err = sc.deleteSecretRecord(namespace, name) //todo delete it from database
			if err != nil {
				beego.Error("delete secret from database failed: ", err)
			}
		}
		return err
	}

	return sc.syncSecretRecord(*s)
}
