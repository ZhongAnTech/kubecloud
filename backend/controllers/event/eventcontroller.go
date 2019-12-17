/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package event

import (
	"fmt"
	"github.com/astaxie/beego"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"

	"k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type EventController struct {
	cluster    string
	kubeClient kubernetes.Interface

	syncHandler  func(eventKey string) (bool, error)
	enqueueEvent func(event *v1.Event)

	eventSynced cache.InformerSynced

	eventList corelisters.EventLister

	queue workqueue.RateLimitingInterface
}

func NewEventController(cluster string, eventInformer coreinformers.EventInformer, kubeClient kubernetes.Interface) *EventController {
	ec := &EventController{
		cluster:    cluster,
		kubeClient: kubeClient,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "event"),
	}

	eventInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ec.addEvent,
		UpdateFunc: ec.updateEvent,
		DeleteFunc: ec.deleteEvent,
	})
	ec.syncHandler = ec.syncEvent
	ec.enqueueEvent = ec.enqueue

	ec.eventList = eventInformer.Lister()
	ec.eventSynced = eventInformer.Informer().HasSynced

	return ec
}

func (ec *EventController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer ec.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("event", stopCh, ec.eventSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(ec.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (ec *EventController) worker() {
	for ec.processNextWorkItem() {
	}
}

func (ec *EventController) processNextWorkItem() bool {
	key, quit := ec.queue.Get()
	if quit {
		return false
	}
	defer ec.queue.Done(key)

	forget, err := ec.syncHandler(key.(string))
	if err == nil {
		if forget {
			ec.queue.Forget(key)
		}
		return true
	}

	// utilruntime.HandleError(fmt.Errorf("Error syncing event: %v", err))
	ec.queue.AddRateLimited(key)

	return true
}

func (ec *EventController) enqueue(event *v1.Event) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(event)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", event, err))
		return
	}

	ec.queue.Add(key)
}

func (ec *EventController) addEvent(obj interface{}) {
	event := obj.(*v1.Event)
	// beego.Debug("Adding Event", ec.cluster, event.Namespace, event.Name)
	ec.enqueueEvent(event)
}

func (ec *EventController) updateEvent(old, cur interface{}) {
	// olding := old.(*v1.Event)
	curing := cur.(*v1.Event)
	// beego.Debug("Updating Event", ec.cluster, olding.Namespace, olding.Name)
	ec.enqueueEvent(curing)
}

func (ec *EventController) deleteEvent(obj interface{}) {
	event, ok := obj.(*v1.Event)

	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		event, ok = tombstone.Obj.(*v1.Event)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a event %+v", obj))
			return
		}
	}
	// beego.Debug("Deleting Event", ec.cluster, event.Namespace, event.Name)
	ec.enqueueEvent(event)
}

func (ec *EventController) syncEvent(key string) (bool, error) {
	// startTime := time.Now()
	// defer func() {
	// 	beego.Info(fmt.Sprintf("Finished syncing event %q (%v)", key, time.Now().Sub(startTime)))
	// }()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// beego.Error(fmt.Sprintf("Error syncing event: %v", err.Error()))
		return false, err
	}
	if len(ns) == 0 || len(name) == 0 {
		return false, fmt.Errorf("invalid event key %q: either namespace or name is missing", key)
	}
	event, err := ec.eventList.Events(ns).Get(name)
	if err != nil {
		// beego.Error(fmt.Sprintf("Error syncing event: %v", err.Error()))
		return false, err
	}

	return ec.syncEventRecord(event)
}

func (ec *EventController) syncEventRecord(event *v1.Event) (bool, error) {
	firstTime, _ := time.Parse("2006-01-02 15:04:05", event.FirstTimestamp.Local().Format("2006-01-02 15:04:05"))
	lastTime, _ := time.Parse("2006-01-02 15:04:05", event.LastTimestamp.Local().Format("2006-01-02 15:04:05"))
	eventModel := models.ZcloudEvent{
		EventUid:        string(event.ObjectMeta.UID),
		EventType:       event.Type,
		Cluster:         ec.cluster,
		Namespace:       event.ObjectMeta.Namespace,
		SourceComponent: event.Source.Component,
		SourceHost:      event.Source.Host,
		ObjectKind:      event.InvolvedObject.Kind,
		ObjectName:      event.InvolvedObject.Name,
		ObjectUid:       string(event.InvolvedObject.UID),
		FieldPath:       event.InvolvedObject.FieldPath,
		Reason:          event.Reason,
		Message:         event.Message,
		Count:           event.Count,
		FirstTimestamp:  firstTime,
		LastTimestamp:   lastTime,
	}

	if err := dao.CreateEvent(eventModel); err != nil {
		beego.Error(fmt.Sprintf("event object write db has error: %v", err))
	}
	return true, nil
}
