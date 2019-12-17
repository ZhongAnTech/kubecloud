package resource

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"
	"kubecloud/common/validate"
)

const (
	DepartmentLabel string = "department"
	BizClusterLabel string = "bizcluster"

	NodeCertificateStartAt = "node.certificate.validity/start"
	NodeCertificateEndAt   = "node.certificate.validity/end"

	NodeRoleMaster = "master"
	NodeRoleNode   = "node"
)

type NodeInfo struct {
	IP       string            `json:"ip"`
	Labels   map[string]string `json:"labels"`
	Status   string            `json:"status"`
	Types    []string          `json:"types"`
	External bool              `json:"external"`
}

type NodeCreate struct {
	Nodes      []NodeInfo `json:"node"`
	DeployMode string     `json:"deploy_mode"`
}

func (create *NodeCreate) Verify(cluster string) error {
	c, err := dao.GetCluster(cluster)
	if err != nil {
		return err
	}
	for i, node := range create.Nodes {
		for j, item := range create.Nodes {
			if i != j && node.IP == item.IP {
				return fmt.Errorf("%s has been duplicated in your node list, please check!", node.IP)
			}
		}
		if dao.CheckNodeExist(cluster, node.IP) {
			return fmt.Errorf("%s is existed\n", node.IP)
		}
		if net.ParseIP(node.IP) == nil {
			return fmt.Errorf("%s format is not right\n", node.IP)
		}

		newLabels := map[string]string{}
		for k, v := range node.Labels {
			if strings.HasPrefix(k, c.LabelPrefix) {
				k = strings.TrimPrefix(k, c.LabelPrefix)
			}
			if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
				continue
			}
			if err := validate.ValidateString(k); err != nil {
				return fmt.Errorf("format of label key %s is not right,%s\n", k, err.Error())
			}
			if err := validate.ValidateString(v); err != nil {
				return fmt.Errorf("format of label value %s is not right,%s\n", v, err.Error())
			}

			newLabels[c.LabelPrefix+k] = v
		}
		node.Labels = newLabels
	}
	return nil
}

type NodeUpdate struct {
	Cluster     string            `json:"cluster"`
	Status      string            `json:"status"`
	DeployError string            `json:"deploy_err"`
	Labels      map[string]string `json:"labels"`
}

func (node *NodeUpdate) Verify() error {
	c, err := dao.GetCluster(node.Cluster)
	if err != nil {
		return err
	}
	newLabels := map[string]string{}
	for k, v := range node.Labels {
		if strings.HasPrefix(k, c.LabelPrefix) {
			k = strings.TrimPrefix(k, c.LabelPrefix)
		}
		newLabels[c.LabelPrefix+k] = v
	}
	err = validate.ValidateLabels(keyword.K8S_RESOURCE_TYPE_NODE, newLabels)
	if err != nil {
		return err
	}
	node.Labels = newLabels
	return nil
}

type Node struct {
	ID                    int64             `json:"id"`
	Name                  string            `json:"name"`
	Cluster               string            `json:"cluster"`
	Department            string            `json:"department"`
	BizCluster            string            `json:"bizcluster"`
	IP                    string            `json:"ip"`
	PodCidr               string            `json:"pod_cidr"`
	Labels                map[string]string `json:"labels"`
	CPU                   float64           `json:"cpu"`
	CPURequests           float64           `json:"cpu_requests"`
	CPULimits             float64           `json:"cpu_limits"`
	CPURequestsPercent    int64             `json:"cpu_requests_percent"`
	Memory                int64             `json:"memory"`
	MemoryRequests        int64             `json:"memory_requests"`
	MemoryLimits          int64             `json:"memory_limits"`
	MemoryRequestsPercent int64             `json:"memory_requests_percent"`
	Creator               string            `json:"creator"`
	CreateAt              string            `json:"create_at"`
	Status                string            `json:"status"`
	CertDeadline          string            `json:"cert_deadline"`
	CertValidity          string            `json:"cert_validity"`
	DeployMode            string            `json:"deploy_mode"`
	Role                  string            `json:"role"`
	Monitor               bool              `json:"monitor"`
	Influxdb              bool              `json:"influxdb"`
	Ingress               bool              `json:"ingress"`
	External              bool              `json:"external"`
}

type NodeDetail struct {
	Node
	SystemInfo *corev1.NodeSystemInfo `json:"systeminfo"`
}

type NodeFreeze struct {
	DeletePods bool `json:"pod_delete"`
}

type NodeDeleteForCMDB struct {
	IP       string `json:"ip"`
	Platform string `json:"platform"`
}

type NodeCreateForCMDB struct {
	Name       string `json:"name"`
	Cluster    string `json:"cluster"`
	Department string `json:"department"`
	BizCluster string `json:"bizcluster"`
	IP         string `json:"ip"`
	PodCidr    string `json:"pod_cidr"`
	Cpu        int64  `json:"cpu"`
	Memory     int64  `json:"memory"`
	Platform   string `json:"platform"`
}

func CreateNode(node Node) error {
	labels, err := json.Marshal(node.Labels)
	if err != nil {
		return fmt.Errorf("the labels of node %s can not be parsed, %s\n", node.IP, err.Error())
	}
	nodeModel := models.ZcloudNode{
		Name:           node.Name,
		Cluster:        node.Cluster,
		Department:     node.Department,
		BizCluster:     node.BizCluster,
		IP:             node.IP,
		Labels:         string(labels),
		Status:         models.NodeStatusPending,
		Creator:        node.Creator,
		HardAddons:     models.NewHardAddons(),
		BizLabelStatus: models.BizLabelSetWaiting,
		DeployMode:     node.DeployMode,
		Monitor:        node.Monitor,
		Influxdb:       node.Influxdb,
		Ingress:        node.Ingress,
		External:       node.External,
	}

	return dao.CreateNode(nodeModel)
}

func UpdateNode(cluster, name string, nodeUpdate NodeUpdate) error {
	labels, err := json.Marshal(nodeUpdate.Labels)
	if err != nil {
		return err
	}

	item, err := dao.GetNodeByName(cluster, name)
	if err != nil {
		return err
	}

	//if item.Status != models.NodeStatusRunning {
	//	//kube-deploy will pass the error info of deployment to zcloud
	//	if item.DeployMode == models.DeployAuto {
	//		item.Status = nodeUpdate.Status
	//		item.DeployError = nodeUpdate.DeployError
	//		if err := dao.UpdateNode(*item); err != nil {
	//			return fmt.Errorf("update node in db has error: %v", err)
	//		}
	//		return nil
	//	}
	//	return fmt.Errorf("update node failed, status is not running")
	//}

	// overwrite node labels in k8s
	if item.Status == models.NodeStatusRunning {
		client, err := service.GetClientset(cluster)
		if err != nil {
			return fmt.Errorf("get client error: %v", err)
		}
		node, err := client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for key, value := range nodeUpdate.Labels {
			node.ObjectMeta.Labels[key] = value
		}
		if _, err := client.CoreV1().Nodes().Update(node); err != nil {
			return fmt.Errorf("update node in k8s has error: %v", err)
		}
		item.Labels = string(labels)
		if err := dao.UpdateNode(*item); err != nil {
			return fmt.Errorf("update node in db has error: %v", err)
		}
	}

	return nil
}

func FreezeNode(cluster, name string, deletePods bool) error {
	nodeModel, err := dao.GetNodeByName(cluster, name)
	if err != nil {
		return err
	}
	if nodeModel.Status != models.NodeStatusRunning {
		return fmt.Errorf("only running node can be freeze, now status is %s", nodeModel.Status)
	}

	// update node in k8s
	client, err := service.GetClientset(cluster)
	if err != nil {
		return fmt.Errorf("get client error: %v", err)
	}

	// set node to unschedulable
	node, err := client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !node.Spec.Unschedulable {
		node.Spec.Unschedulable = true

		if _, err := client.CoreV1().Nodes().Update(node); err != nil {
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

		podList, err := client.CoreV1().Pods(corev1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		})
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			if pod.Namespace == "kube-system" || pod.Namespace == "kube-public" || pod.Namespace == "istio-system" {
				continue
			}
			if err := client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}

	nodeModel.Status = models.NodeStatusOffline

	return dao.UpdateNode(*nodeModel)
}

func UnfreezeNode(cluster, name string) error {
	nodeModel, err := dao.GetNodeByName(cluster, name)
	if err != nil {
		return err
	}
	if nodeModel.Status != models.NodeStatusOffline {
		return fmt.Errorf("only offline node can be unfreeze, now status is %s", nodeModel.Status)
	}

	// update node in k8s
	client, err := service.GetClientset(cluster)
	if err != nil {
		return fmt.Errorf("get client error: %v", err)
	}
	node, err := client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if node.Spec.Unschedulable {
		node.Spec.Unschedulable = false

		if _, err := client.CoreV1().Nodes().Update(node); err != nil {
			return fmt.Errorf("update node in k8s has error: %v", err)
		}
	}

	nodeModel.Status = models.NodeStatusRunning

	return dao.UpdateNode(*nodeModel)
}

func DeleteNode(cluster, name string) error {
	nodeModel, err := dao.GetNodeByName(cluster, name)
	if err != nil {
		return err
	}
	if nodeModel.Status == models.NodeStatusRunning {
		return fmt.Errorf("node can be delete, if the node is running, you can freeze it firstly!")
	}
	// delete node in k8s
	client, err := service.GetClientset(cluster)
	if err != nil {
		return fmt.Errorf("get client error: %v", err.Error())
	}
	if err := client.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{}); err != nil {
		beego.Error(fmt.Sprintf("Delete node error: %v", err.Error()))
		return err
	}
	return dao.DeleteNode(cluster, name)
}

func DeleteNodeSoft(cluster, name string) error {
	err := dao.DeleteNode(cluster, name)
	if err != nil {
		return fmt.Errorf("Delete node from database failed: %v", err)
	}
	return nil
}

//func NodeCreateNotify(node *models.ZcloudNode) error {
//	if cmdbServer != "" {
//		beego.Info("NodeCreateNotify notify cmdb begin:", node.Cluster, node.IP)
//		cmdbInfo := NodeCreateForCMDB{}
//		cmdbInfo.Name = node.Name
//		cmdbInfo.Cluster = node.Cluster
//		cmdbInfo.Department = node.Department
//		cmdbInfo.BizCluster = node.BizCluster
//		cmdbInfo.IP = node.IP
//		cmdbInfo.PodCidr = node.PodCidr
//		cmdbInfo.Cpu = node.CPU
//		cmdbInfo.Memory = node.Memory
//		cmdbInfo.Platform = cmdbPlatform
//		cmdbInfoBytes, err := json.Marshal(cmdbInfo)
//		if err != nil {
//			return fmt.Errorf("marshal cmdb info failed:%s, %s", node.Cluster, node.IP)
//		}
//
//		cmdbUrl := fmt.Sprintf("%s/api/k8s/node/add", cmdbServer)
//
//		if _, err := SentRequestToCmdb("POST", cmdbUrl, bytes.NewReader(cmdbInfoBytes)); err != nil {
//			return fmt.Errorf("notify node create info to cmdb failed: %v", err)
//		}
//		beego.Info("NodeCreateNotify notify cmdb end:", node.Cluster, node.IP)
//	}
//	return nil
//}
//
//func NodeDeleteNotify(cluster, ip string) error {
//	if cmdbServer != "" {
//		beego.Info("NodeDeleteNotify notify cmdb begin:", cluster, ip)
//		cmdbInfo := NodeDeleteForCMDB{}
//		cmdbInfo.IP = ip
//		cmdbInfo.Platform = cmdbPlatform
//
//		cmdbInfoBytes, err := json.Marshal(cmdbInfo)
//		if err != nil {
//			return fmt.Errorf("marshal cmdb info failed:%s, %s", cluster, ip)
//		} else {
//			cmdbUrl := fmt.Sprintf("%s/api/k8s/app/down_res", cmdbServer)
//			if _, err = SentRequestToCmdb("POST", cmdbUrl, bytes.NewReader(cmdbInfoBytes)); err != nil {
//				return fmt.Errorf("notify node delete info to cmdb failed:%s, %s", cluster, err)
//			}
//			beego.Info("NodeDeleteNotify notify cmdb end:", cluster, ip)
//		}
//	}
//	return nil
//}
//
//func NodeNotifyCmdb(node *models.ZcloudNode, newBizCluster, oldBizCluster string) error {
//	if newBizCluster != oldBizCluster {
//		if newBizCluster != ""{
//			if err := NodeCreateNotify(node); err != nil {
//				return fmt.Errorf("node create notify cmdb error:%v", err)
//			}
//		} else {
//			if err := NodeDeleteNotify(node.Cluster, node.IP); err != nil {
//				return fmt.Errorf("node delete notify cmdb error:%v", err)
//			}
//		}
//	}
//	return nil
//}

func GetNodeListFilter(cluster string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	nodes := []Node{}
	res, err := dao.GetNodeList(cluster, filterQuery)
	if err != nil {
		return nil, err
	}
	list, ok := res.List.([]*models.ZcloudNode)
	if !ok {
		return nil, fmt.Errorf("data type is not right!")
	}
	for _, item := range list {
		var labels map[string]string
		if err := json.Unmarshal([]byte(item.Labels), &labels); err != nil {
			beego.Error(fmt.Sprintf("node labels json unmarshal error: %v", err))
		}
		nodeRole := NodeRoleNode
		if _, exist := labels["node-role.kubernetes.io/master"]; exist {
			nodeRole = NodeRoleMaster
		}
		//cpu单位转化为核
		nodeCpu, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPU)/1000), 64)
		nodeCPURequests, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPURequests)/1000), 64)
		nodeCPULimits, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPULimits)/1000), 64)

		var nodeCPURequestsPercent, nodeMemoryRequestsPercent int64
		if item.CPU != 0 {
			nodeCPURequestsPercent = 100 * item.CPURequests / item.CPU
		}

		if item.Memory != 0 {
			nodeMemoryRequestsPercent = 100 * item.MemoryRequests / item.Memory
		}

		nodes = append(nodes, Node{
			ID:                 item.Id,
			Name:               item.Name,
			Cluster:            item.Cluster,
			Department:         item.Department,
			BizCluster:         item.BizCluster,
			IP:                 item.IP,
			PodCidr:            item.PodCidr,
			Labels:             labels,
			CPU:                nodeCpu,
			CPURequests:        nodeCPURequests,
			CPULimits:          nodeCPULimits,
			CPURequestsPercent: nodeCPURequestsPercent,
			//内存单位转化为Mi
			Memory:                item.Memory / 1024 / 1024,
			MemoryRequests:        item.MemoryRequests / 1024 / 1024,
			MemoryLimits:          item.MemoryLimits / 1024 / 1024,
			MemoryRequestsPercent: nodeMemoryRequestsPercent,
			CreateAt:              item.CreateAt.Format("2006-01-02 15:04:05"),
			Status:                item.Status,
			Role:                  nodeRole,
		})
	}
	res.List = nodes
	return res, nil
}

// func GetNodeList(cluster, bizcluster string) (*NodeList, error) {
// 	nodes := []Node{}
// 	items, err := dao.GetNodeList(cluster, bizcluster)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, item := range items {
// 		var labels map[string]string
// 		if err := json.Unmarshal([]byte(item.Labels), &labels); err != nil {
// 			beego.Error(fmt.Sprintf("node labels json unmarshal error: %v", err))
// 		}
// 		nodes = append(nodes, Node{
// 			ID:             item.Id,
// 			Name:           item.Name,
// 			Cluster:        item.Cluster,
// 			BizCluster:     item.BizCluster,
// 			IP:             item.IP,
// 			PodCidr:        item.PodCidr,
// 			Labels:         labels,
// 			CPU:            item.CPU,
// 			CPURequests:    item.CPURequests,
// 			CPULimits:      item.CPULimits,
// 			Memory:         item.Memory,
// 			MemoryRequests: item.MemoryRequests,
// 			MemoryLimits:   item.MemoryLimits,
// 			CreateAt:       item.CreateAt.Format("2006-01-02 15:04:05"),
// 			Status:         item.Status,
// 		})
// 	}

// 	return &NodeList{Nodes: nodes}, nil
// }

func GetNodeDetail(cluster, nodeName string) (*NodeDetail, error) {
	item, err := dao.GetNodeByName(cluster, nodeName)
	if err != nil {
		return nil, err
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(item.Labels), &labels); err != nil {
		beego.Error(fmt.Sprintf("node labels json unmarshal error: %v", err))
	}
	//cpu单位转化为核
	nodeCpu, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPU)/1000), 64)
	nodeCPURequests, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPURequests)/1000), 64)
	nodeCPULimits, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", float64(item.CPULimits)/1000), 64)

	var nodeCPURequestsPercent, nodeMemoryRequestsPercent int64
	if item.CPU != 0 {
		nodeCPURequestsPercent = 100 * item.CPURequests / item.CPU
	}

	if item.Memory != 0 {
		nodeMemoryRequestsPercent = 100 * item.MemoryRequests / item.Memory
	}

	node := Node{
		ID:                 item.Id,
		Name:               item.Name,
		Cluster:            item.Cluster,
		BizCluster:         item.BizCluster,
		IP:                 item.IP,
		PodCidr:            item.PodCidr,
		Labels:             labels,
		CPU:                nodeCpu,
		CPURequests:        nodeCPURequests,
		CPULimits:          nodeCPULimits,
		CPURequestsPercent: nodeCPURequestsPercent,
		//内存单位转化为Mi
		Memory:                item.Memory / 1024 / 1024,
		MemoryRequests:        item.MemoryRequests / 1024 / 1024,
		MemoryLimits:          item.MemoryLimits / 1024 / 1024,
		MemoryRequestsPercent: nodeMemoryRequestsPercent,
		Creator:               item.Creator,
		CreateAt:              item.CreateAt.Format("2006-01-02 15:04:05"),
		Status:                item.Status,
	}

	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("get client error %v", err)
	}
	k8sNode, err := client.CoreV1().Nodes().Get(item.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	deadline, err := GetCertificateTime(k8sNode.Annotations, NodeCertificateEndAt)
	if err != nil {
		node.CertDeadline = "unkown"
		node.CertValidity = "unkown"
	} else {
		deadline = deadline.Local()
		node.CertDeadline = deadline.Format("2006-01-02 15:04:05")
		now := time.Now()
		if deadline.Sub(now) <= 0 {
			node.CertValidity = "expired"
		} else {
			durationHours := int(deadline.Sub(now).Hours())
			days := durationHours / 24
			hours := durationHours - days*24
			node.CertValidity = fmt.Sprintf("%vDays%vHours", days, hours)
		}
	}
	nodeSysteminfo := k8sNode.Status.NodeInfo

	// nodeImages := []NodeImage{}
	// for _, img := range k8sNode.Status.Images {
	// 	nodeImages = append(nodeImages, NodeImage{
	// 		Names:      img.Names,
	// 		SizeMBytes: int64(math.Ceil(float64(img.SizeBytes) / 1048576.0)),
	// 	})
	// }

	return &NodeDetail{
		Node:       node,
		SystemInfo: &nodeSysteminfo,
		// NodeImages: nodeImages,
	}, nil
}

func GetCertificateTime(annotations map[string]string, key string) (time.Time, error) {
	if value, ok := annotations[key]; ok {
		time, err := time.Parse("2006-01-02 15:04:05", value)
		if err != nil {
			beego.Warn("parse time failed:", err)
		}
		return time, nil
	}
	return time.Now(), fmt.Errorf("unkown time!")
}

type NodeEvent struct {
	EventLevel   string `json:"event_level"`
	EventObject  string `json:"event_object"`
	EventType    string `json:"event_type"`
	EventMessage string `json:"event_message"`
	EventTime    string `json:"event_time"`
}

func GetNodeEvent(cluster, nodeName string, filter *utils.FilterQuery) (*utils.QueryResult, error) {
	node, err := dao.GetNodeByName(cluster, nodeName)
	if err != nil {
		return nil, err
	}
	eventList, err := dao.GetNodeEventsByFilter(cluster, node.Name, filter)
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

func GetSimpleNodes(cluster string) ([]corev1.Node, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("get client error %v", err)
	}

	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	labelKey := beego.AppConfig.String("prometheus::serviceNodeLabelKey")
	labelValue := beego.AppConfig.String("prometheus::serviceNodeLabelValue")

	nodes := []corev1.Node{}
	for _, node := range nodeList.Items {
		value, ok := node.Labels[labelKey]
		if ok && value == labelValue {
			//labels, _ := json.Marshal(node.Labels)
			//beego.Debug("Select node's labels is: ", string(labels))
			nodes = append(nodes, node)
		}
	}
	if len(nodes) < 1 {
		return nodes, fmt.Errorf("can't find ingress node with label '%s'='%s'", labelKey, labelValue)
	}
	return nodes, nil
}

func GetNodePods(cluster, nodeName string) ([]*Pod, error) {
	node, err := dao.GetNodeByName(cluster, nodeName)
	if err != nil {
		return nil, err
	}

	podList, err := GetPodsByNode(cluster, node.Name)
	if err != nil {
		return nil, err
	}

	pods := []*Pod{}
	for _, ipod := range podList {
		if ipod.Namespace == "kube-system" || ipod.Namespace == "ingress-nginx" {
			continue
		}
		pods = append(pods, ipod)
	}

	return pods, nil
}

func CleanNodes() error {
	clusters, err := dao.GetAllClusters()
	if err != nil {
		return err
	}
	for _, icluster := range clusters {
		client, err := service.GetClientset(icluster.Name)
		if err != nil {
			return fmt.Errorf("get client error %v", err)
		}

		res, err := dao.GetNodeList(icluster.Name, nil)
		if err != nil {
			return err
		}
		nodes := res.List.([]*models.ZcloudNode)
		for _, inode := range nodes {
			if _, err := client.CoreV1().Nodes().Get(inode.Name, metav1.GetOptions{}); err != nil {
				dao.DeleteNode(icluster.Name, inode.Name)
			}
		}
	}
	return nil
}
