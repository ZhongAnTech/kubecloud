package resource

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"

	"kubecloud/backend/dao"
	"kubecloud/backend/service"
	"kubecloud/common/utils"
)

type Node struct {
	cluster string
	client  kubernetes.Interface
}

func NewNode(cluster string) (*Node, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, err
	}
	return &Node{
		cluster: cluster,
		client:  client,
	}, nil
}

func (n *Node) ListNode() (*corev1.NodeList, error) {
	nodeList, err := n.client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList, nil
}

func (n *Node) GetNode(name string) (*corev1.Node, error) {
	node, err := n.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) UpdateNode(name string, labels map[string]string) error {
	node, err := n.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.ObjectMeta.Labels = labels
	if _, err := n.client.CoreV1().Nodes().Update(node); err != nil {
		return err
	}
	return nil
}

func (n *Node) DeleteNode(name string) error {
	if err := n.client.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (n *Node) FreezeNode(name string, deletePods bool) error {
	// set node to unschedulable
	node, err := n.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !node.Spec.Unschedulable {
		node.Spec.Unschedulable = true
		if _, err := n.client.CoreV1().Nodes().Update(node); err != nil {
			return fmt.Errorf("update node in k8s has error: %v", err)
		}
	}
	// delete node of pods
	if deletePods {
		fieldSelector, err := fields.ParseSelector("spec.nodeName=" + name +
			",status.phase!=" + string(corev1.PodSucceeded) +
			",status.phase!=" + string(corev1.PodFailed))
		if err != nil {
			return err
		}
		podList, err := n.client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		})
		if err != nil {
			return err
		}
		for _, pod := range podList.Items {
			if err := n.client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Node) UnfreezeNode(name string) error {
	node, err := n.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if node.Spec.Unschedulable {
		node.Spec.Unschedulable = false

		if _, err := n.client.CoreV1().Nodes().Update(node); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) GetNodePods(name string) (*corev1.PodList, error) {
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + name +
		",status.phase!=" + string(corev1.PodSucceeded) +
		",status.phase!=" + string(corev1.PodFailed))
	if err != nil {
		return nil, err
	}
	pods, err := n.client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

type NodeEvent struct {
	EventLevel   string `json:"event_level"`
	EventObject  string `json:"event_object"`
	EventType    string `json:"event_type"`
	EventMessage string `json:"event_message"`
	EventTime    string `json:"event_time"`
}

func (n *Node) GetNodeEvent(name string, filter *utils.FilterQuery) (*utils.QueryResult, error) {
	eventList, err := dao.GetNodeEventsByFilter(n.cluster, name, filter)
	if err != nil {
		return nil, err
	}
	return eventList, nil
}

type NodeUsedResource struct {
	RequestCPU    int64
	LimitCPU      int64
	RequestMemory int64
	LimitMemory   int64
}

func GetNodeUsedResource(cluster, node string) (*NodeUsedResource, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("get client error %v", err)
	}

	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node +
		",status.phase!=" + string(corev1.PodSucceeded) +
		",status.phase!=" + string(corev1.PodFailed))
	if err != nil {
		return nil, err
	}

	podList, err := client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, err
	}

	reqs, limits := getPodsTotalRequestsAndLimits(podList)
	cpuReqs, cpuLimits, memoryReqs, memoryLimits :=
		reqs[corev1.ResourceCPU], limits[corev1.ResourceCPU], reqs[corev1.ResourceMemory], limits[corev1.ResourceMemory]
	nodeRes := &NodeUsedResource{
		RequestCPU:    cpuReqs.MilliValue(),
		LimitCPU:      cpuLimits.MilliValue(),
		RequestMemory: memoryReqs.Value(),
		LimitMemory:   memoryLimits.Value(),
	}
	// for _, ipod := range podList.Items {
	// 	for _, c := range ipod.Spec.Containers {
	// 		// requests
	// 		nodeRes.RequestCPU += c.Resources.Requests.Cpu().MilliValue()
	// 		nodeRes.RequestMemory += c.Resources.Requests.Memory().Value()
	// 		// limits
	// 		nodeRes.LimitCPU += c.Resources.Limits.Cpu().MilliValue()
	// 		nodeRes.LimitMemory += c.Resources.Limits.Memory().Value()
	// 	}
	// }

	return nodeRes, nil
}

func getPodsTotalRequestsAndLimits(podList *corev1.PodList) (reqs map[corev1.ResourceName]resource.Quantity, limits map[corev1.ResourceName]resource.Quantity) {
	reqs, limits = map[corev1.ResourceName]resource.Quantity{}, map[corev1.ResourceName]resource.Quantity{}
	for _, pod := range podList.Items {
		podReqs, podLimits := PodRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = podReqValue.DeepCopy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod. If pod overhead is non-nil, the pod overhead is added to the
// total container resource requests and to the total container limits which have a
// non-zero quantity.
func PodRequestsAndLimits(pod *corev1.Pod) (reqs, limits corev1.ResourceList) {
	reqs, limits = corev1.ResourceList{}, corev1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		addResourceList(reqs, container.Resources.Requests)
		addResourceList(limits, container.Resources.Limits)
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		maxResourceList(reqs, container.Resources.Requests)
		maxResourceList(limits, container.Resources.Limits)
	}

	// Add overhead for running a pod to the sum of requests and to non-zero limits:
	if pod.Spec.Overhead != nil {
		addResourceList(reqs, pod.Spec.Overhead)

		for name, quantity := range pod.Spec.Overhead {
			if value, ok := limits[name]; ok && !value.IsZero() {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	return
}

// addResourceList adds the resources in newList to list
func addResourceList(list, new corev1.ResourceList) {
	for name, quantity := range new {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}
}

// maxResourceList sets list to the greater of list/newList for every resource
// either list
func maxResourceList(list, new corev1.ResourceList) {
	for name, quantity := range new {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
			continue
		} else {
			if quantity.Cmp(value) > 0 {
				list[name] = quantity.DeepCopy()
			}
		}
	}
}
