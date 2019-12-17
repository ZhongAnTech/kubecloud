package resource

import (
	"fmt"
	"kubecloud/backend/service"

	"github.com/astaxie/beego"

	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

const (
	PodStatusRunning  = "Running"
	PodStatusNotReady = "NotReady"
)

type Pod struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Version      string            `json:"version"`
	NodeIP       string            `json:"node_ip"`
	PodIP        string            `json:"pod_ip"`
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	RestartCount int32             `json:"restart_count"`
	StartTime    string            `json:"start_time"`
	Labels       map[string]string `json:"labels"`
	Containers   []*PodContainer   `json:"containers"`
}

type PodContainer struct {
	Name           string `json:"name"`
	Image          string `json:"image"`
	CpuRequests    int64  `json:"cpu_requests"`
	CpuLimits      int64  `json:"cpu_limits"`
	MemoryRequests int64  `json:"memory_requests"`
	MemoryLimits   int64  `json:"memory_limits"`
}

func getPodStatus(pod v1.Pod) (status string, message string, restartCount int32) {
	status = PodStatusRunning
	if pod.Status.Phase == v1.PodRunning {
		for _, c := range pod.Status.Conditions {
			if c.Type == v1.PodReady {
				if c.Status == v1.ConditionFalse {
					status = PodStatusNotReady
					message = fmt.Sprintf("%s: %s", c.Reason, c.Message)
					break
				}
			}
		}
	} else {
		status = PodStatusNotReady
		for _, c := range pod.Status.Conditions {
			if c.Type == v1.PodScheduled {
				if c.Status != v1.ConditionTrue {
					message = fmt.Sprintf("%s: %s", c.Reason, c.Message)
					break
				}
			}
		}
	}

	for _, cs := range pod.Status.ContainerStatuses {
		restartCount += cs.RestartCount
		if status == PodStatusNotReady && cs.State.Waiting != nil {
			message = fmt.Sprintf("%s: %s", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
		if status == PodStatusNotReady && cs.State.Terminated != nil {
			message = fmt.Sprintf("%s: %s", cs.State.Terminated.Reason, cs.State.Terminated.Message)
		}
	}

	return
}

func podConv(k8sPod v1.Pod) *Pod {
	pod := &Pod{
		Name:      k8sPod.ObjectMeta.Name,
		Namespace: k8sPod.ObjectMeta.Namespace,
		Version:   GetResourceVersion(&k8sPod, ResTypePod, ""),
		NodeIP:    k8sPod.Status.HostIP,
		PodIP:     k8sPod.Status.PodIP,
		Labels:    k8sPod.Labels,
	}
	pod.Status, pod.Message, pod.RestartCount = getPodStatus(k8sPod)
	if k8sPod.Status.StartTime != nil {
		pod.StartTime = k8sPod.Status.StartTime.Format("2006-01-02 15:04:05")
	}

	for _, k8sContainer := range k8sPod.Spec.Containers {
		container := podContainerConv(k8sContainer)
		pod.Containers = append(pod.Containers, container)
	}

	return pod
}

func podContainerConv(k8scontainer v1.Container) *PodContainer {
	container := &PodContainer{
		Name:           k8scontainer.Name,
		Image:          k8scontainer.Image,
		CpuRequests:    k8scontainer.Resources.Requests.Cpu().MilliValue(),
		CpuLimits:      k8scontainer.Resources.Limits.Cpu().MilliValue(),
		MemoryRequests: k8scontainer.Resources.Requests.Memory().Value(),
		MemoryLimits:   k8scontainer.Resources.Limits.Memory().Value(),
	}
	return container
}

func GetPods(cluster, namespace, labelSelector string, replicas int) ([]*Pod, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error(fmt.Sprintf("Get %v pods on namespace %v in cluster %v Error: %v", labelSelector, namespace, cluster, err.Error()))
		return nil, err
	}
	k8sPods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		beego.Error(fmt.Sprintf("Get %v pods on namespace %v in cluster %v Error: %v", labelSelector, namespace, cluster, err.Error()))
		return nil, err
	}
	pods := []*Pod{}
	delPods := []v1.Pod{}
	podNum := len(k8sPods.Items)
	noPodIPNum := 0
	for _, k8spod := range k8sPods.Items {
		if k8spod.Status.PodIP == "" {
			if k8spod.Status.Reason == "Evicted" {
				delPods = append(delPods, k8spod)
				continue
			}
			// if pod number is larger than replicas*2, it is not normal, dont need to check
			added := false
			if podNum > (replicas << 1) {
				noPodIPNum++
				added = true
				if noPodIPNum > (replicas<<1) && replicas != 0 {
					continue
				}
			}
			if isTrashPod(client, cluster, k8spod) {
				// trash pod, filtered
				if added {
					noPodIPNum--
				}
				delPods = append(delPods, k8spod)
				continue
			}
		}
		pod := podConv(k8spod)
		pods = append(pods, pod)
	}
	if len(delPods) > 0 {
		go func(pods []v1.Pod, selector string) {
			beego.Warn(fmt.Sprintf("%v trash or evicted pods about %v will be deleted!", len(delPods), selector))
			for _, pod := range pods {
				gracePeriod := int64(0)
				client.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
				})
			}
			beego.Warn(fmt.Sprintf("%v trash or evicted pods about %v have been deleted!", len(delPods), selector))
		}(delPods, labelSelector)
	}
	return pods, nil
}

func GetPodsByNode(cluster, node string) ([]*Pod, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, err
	}
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node +
		",status.phase!=" + string(v1.PodSucceeded) +
		",status.phase!=" + string(v1.PodFailed))
	if err != nil {
		return nil, err
	}
	k8sPods, err := client.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	pods := []*Pod{}
	for _, k8spod := range k8sPods.Items {
		pod := podConv(k8spod)
		pods = append(pods, pod)
	}
	return pods, nil
}

func isTrashPod(client kubernetes.Interface, cluster string, pod v1.Pod) bool {
	ref := metav1.GetControllerOf(&pod)
	if ref == nil {
		return true
	}
	switch ref.Kind {
	case extensions.SchemeGroupVersion.WithKind("ReplicaSet").Kind:
		return NewDeploymentRes(client, cluster, pod.Namespace).GetOwnerForPod(pod, ref) == nil
	}
	return false
}
