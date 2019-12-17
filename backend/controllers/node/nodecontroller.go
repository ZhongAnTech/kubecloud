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

package node

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"

	"github.com/astaxie/beego"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type NodeController struct {
	cluster     string
	labelPrefix string
	kubeClient  kubernetes.Interface

	syncHandler func(nodeKey string) (bool, error)
	enqueueNode func(node *v1.Node)

	nodeSynced cache.InformerSynced

	nodeList corelisters.NodeLister

	queue workqueue.RateLimitingInterface
}

func NewNodeController(cluster string, nodeInformer coreinformers.NodeInformer, kubeClient kubernetes.Interface) (*NodeController, error) {
	c, err := dao.GetCluster(cluster)
	if err != nil {
		return nil, err
	}
	nc := &NodeController{
		cluster:     cluster,
		kubeClient:  kubeClient,
		labelPrefix: c.LabelPrefix,
		queue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "node"),
	}
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nc.addEvent,
		UpdateFunc: nc.updateEvent,
		DeleteFunc: nc.deleteEvent,
	})
	nc.syncHandler = nc.syncNode
	nc.enqueueNode = nc.enqueue

	nc.nodeList = nodeInformer.Lister()
	nc.nodeSynced = nodeInformer.Informer().HasSynced

	return nc, nil
}

func (nc *NodeController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer nc.queue.ShutDown()

	if !cache.WaitForNamedCacheSync("node", stopCh, nc.nodeSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(nc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (nc *NodeController) worker() {
	for nc.processNextWorkItem() {
	}
}

func (nc *NodeController) processNextWorkItem() bool {
	key, quit := nc.queue.Get()
	if quit {
		return false
	}
	defer nc.queue.Done(key)

	forget, err := nc.syncHandler(key.(string))
	if err == nil {
		if forget {
			nc.queue.Forget(key)
		}
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing node: %v", err))
	nc.queue.AddRateLimited(key)

	return true
}

func (nc *NodeController) enqueue(node *v1.Node) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(node)
	if err != nil {
		beego.Error(fmt.Errorf("Couldn't get key for object %#v: %v", node, err))
		return
	}

	nc.queue.Add(key)
}

func (nc *NodeController) addEvent(obj interface{}) {
	node := obj.(*v1.Node)
	nc.enqueueNode(node)
}

func (nc *NodeController) updateEvent(old, cur interface{}) {
	// olding := old.(*v1.Node)
	curing := cur.(*v1.Node)
	nc.enqueueNode(curing)
}

func (nc *NodeController) deleteEvent(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		node, ok = tombstone.Obj.(*v1.Node)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a node %+v", obj))
			return
		}
	}
	nc.enqueueNode(node)
}

func (nc *NodeController) syncNode(key string) (bool, error) {
	node, err := nc.nodeList.Get(key)
	if errors.IsNotFound(err) {
		beego.Debug(fmt.Sprintf("Node %v has been deleted, cluster: %s", key, nc.cluster))
		err = resource.DeleteNodeSoft(nc.cluster, key)
		if err != nil {
			beego.Error("Delete node from database failed: ", err)
		}
		return false, err
	}
	if err != nil {
		beego.Error(fmt.Sprintf("Error syncing node: %v", err.Error()))
		return false, err
	}
	return nc.syncNodeRecord(node)
}

const (
	NodeStatusPending string = "Pending"
	NodeStatusRunning string = "Running"
	NodeStatusOffline string = "Offline"
	NodeStatusError   string = "Error"
)

func (nc *NodeController) syncNodeRecord(node *v1.Node) (bool, error) {
	labels := map[string]string{}
	for k, v := range node.Labels {
		if strings.HasPrefix(k, nc.labelPrefix) {
			labels[k] = v
		}
	}
	labelStr, _ := json.Marshal(labels)
	status := NodeStatusRunning
	if node.Spec.Unschedulable {
		status = NodeStatusOffline
	} else {
		for _, cd := range node.Status.Conditions {
			switch cd.Type {
			case v1.NodeReady:
				if cd.Status != v1.ConditionTrue {
					status = NodeStatusError
				}
				break
			}
		}
	}

	nodeRes, err := resource.GetNodeUsedResource(nc.cluster, node.Name)
	if err != nil {
		beego.Error(fmt.Sprintf("sync node error: %v", err))
		return false, err
	}

	ip := ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			ip = addr.Address
		}
	}
	if ip == "" {
		if _, ok := node.ObjectMeta.Annotations["alpha.kubernetes.io/provided-node-ip"]; ok {
			ip = node.ObjectMeta.Annotations["alpha.kubernetes.io/provided-node-ip"]
		} else if _, ok := node.ObjectMeta.Annotations["flannel.alpha.coreos.com/public-ip"]; ok {
			ip = node.ObjectMeta.Annotations["flannel.alpha.coreos.com/public-ip"]
		}
	}

	department := ""
	bizCluster := ""
	if node.Labels != nil {
		department = node.Labels[nc.labelPrefix+resource.DepartmentLabel]
		bizCluster = node.Labels[nc.labelPrefix+resource.BizClusterLabel]
	}

	nodeModel := models.ZcloudNode{
		Name:           node.Name,
		Cluster:        nc.cluster,
		Department:     department,
		BizCluster:     bizCluster,
		IP:             ip,
		PodCidr:        node.Spec.PodCIDR,
		Labels:         string(labelStr),
		CPU:            node.Status.Allocatable.Cpu().MilliValue(),
		CPURequests:    nodeRes.RequestCPU,
		CPULimits:      nodeRes.LimitCPU,
		Memory:         node.Status.Allocatable.Memory().Value(),
		MemoryRequests: nodeRes.RequestMemory,
		MemoryLimits:   nodeRes.LimitMemory,
		Status:         status,
		HardAddons:     models.NewHardAddons(),
	}

	if status == NodeStatusError {
		fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name +
			",status.phase!=" + string(v1.PodUnknown))
		if err != nil {
			beego.Error(err.Error())
			return false, err
		}
		pods, err := nc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		})
		if err != nil {
			beego.Error(err.Error())
			return false, err
		}
		for _, ipod := range pods.Items {
			var gracePeriod int64
			gracePeriod = 0
			err := nc.kubeClient.CoreV1().Pods(ipod.Namespace).Delete(ipod.Name, &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
			if err != nil {
				beego.Error(err.Error())
				return false, err
			}
			beego.Debug(fmt.Sprintf("delete pod %v in node %v: %v", ipod.Name, node.Name, status))
		}
	}
	if err := nc.syncNodeInfo(nodeModel, node, nc.labelPrefix); err != nil {
		beego.Error(fmt.Sprintf("node object write db has error: %v", err))
	}

	return true, nil
}

func (nc *NodeController) syncNodeInfo(node models.ZcloudNode, kubeNode *v1.Node, labelPreffix string) error {
	oldNode, err := dao.GetNodeByName(node.Cluster, node.Name)
	node.CreateAt, _ = time.Parse("2006-01-02 15:04:05", kubeNode.CreationTimestamp.Format("2006-01-02 15:04:05"))
	if err == nil {
		node.Id = oldNode.Id
		node.DeployMode = oldNode.DeployMode
		node.Monitor = oldNode.Monitor
		node.Ingress = oldNode.Ingress
		node.Influxdb = oldNode.Influxdb
		node.Creator = oldNode.Creator
		if oldNode.BizLabelStatus == models.BizLabelSetWaiting {
			node.BizLabelStatus = oldNode.BizLabelStatus
			node.Department = oldNode.Department
			node.BizCluster = oldNode.BizCluster
			if node.Status == NodeStatusRunning {
				if oldNode.BizCluster != "" || oldNode.Department != "" {
					//set bizcluster label
					if kubeNode.ObjectMeta.Labels == nil {
						kubeNode.ObjectMeta.Labels = map[string]string{}
					}
					if oldNode.BizCluster != "" {
						kubeNode.ObjectMeta.Labels[labelPreffix+resource.BizClusterLabel] = oldNode.BizCluster
					}
					if oldNode.Department != "" {
						kubeNode.ObjectMeta.Labels[labelPreffix+resource.DepartmentLabel] = oldNode.Department
					}
					if _, err := nc.kubeClient.CoreV1().Nodes().Update(kubeNode); err != nil {
						return fmt.Errorf("set bizcluster or department label on node in k8s has error: %v", err)
					}
				}
				node.BizLabelStatus = models.BizLabelSetFinished
			}
		}
		err = dao.UpdateNode(node)
	} else {
		err = dao.CreateNode(node)
	}
	return err
}
