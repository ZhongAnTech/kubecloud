package models

const (
	NodeStatusPending string = "Pending"
	NodeStatusRunning string = "Running"
	NodeStatusOffline string = "Offline"
	NodeStatusError   string = "Error"
)

type ZcloudNode struct {
	Id             int64  `orm:"pk;column(id);auto" json:"id"`
	Name           string `orm:"column(name)" json:"name"`
	Cluster        string `orm:"column(cluster)" json:"cluster"`
	Department     string `orm:"column(department)" json:"department"`
	BizCluster     string `orm:"column(bizcluster)" json:"bizcluster"`
	IP             string `orm:"column(ip)" json:"ip"`
	PodCidr        string `orm:"column(pod_cidr)" json:"pod_cidr"`
	Labels         string `orm:"column(labels);size(2048)" json:"labels"`
	CPU            int64  `orm:"column(cpu)" json:"cpu"`
	CPURequests    int64  `orm:"column(cpu_requests)" json:"cpu_requests"`
	CPULimits      int64  `orm:"column(cpu_limits)" json:"cpu_limits"`
	Memory         int64  `orm:"column(memory)" json:"memory"`
	MemoryRequests int64  `orm:"column(memory_requests)" json:"memory_requests"`
	MemoryLimits   int64  `orm:"column(memory_limits)" json:"memory_limits"`
	Status         string `orm:"column(status)" json:"status"`
	InitStatus     string `orm:"column(init_status)" json:"init_status"`
	Creator        string `orm:"column(creator)" json:"creator"`
	BizLabelStatus string `orm:"column(biz_label_status)" json:"biz_label_status"`
	DeployMode     string `orm:"column(deploy_mode)" json:"deploy_mode"`
	DeployError    string `orm:"column(deploy_err)" json:"deploy_err"`
	Monitor        bool   `orm:"column(monitor);default(false)" json:"monitor"`
	Influxdb       bool   `orm:"column(influxdb);default(false)" json:"influxdb"`
	Ingress        bool   `orm:"column(ingress);default(false)" json:"ingress"`
	External       bool   `orm:"column(external);default(false)" json:"external"`
	HardAddons
}

const (
	BizLabelSetWaiting  string = "waiting"
	BizLabelSetFinished string = ""
)

func (u *ZcloudNode) TableUnique() [][]string {
	return [][]string{
		[]string{"Name", "Cluster"},
	}
}

func (t *ZcloudNode) TableName() string {
	return "zcloud_node"
}
