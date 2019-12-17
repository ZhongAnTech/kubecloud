package models

type ZcloudVersion struct {
	Id                int64  `orm:"pk;column(id);auto" json:"id"`
	Cluster           string `orm:"column(cluster)" json:"cluster"`
	Namespace         string `orm:"column(namespace)" json:"namespace"`
	Name              string `orm:"column(name)" json:"name"` // name is application name// name is application name
	Kind              string `orm:"column(kind)" json:"kind"`
	Version           string `orm:"column(version)" json:"version"`
	PodVersion        string `orm:"column(pod_version)" json:"pod_version"`
	Weight            int    `orm:"column(weight)" json:"weight"`
	Stage             string `orm:"column(stage)" json:"stage"` // new or normal
	Replicas          int    `orm:"column(replicas)" json:"replicas"`
	CurReplicas       int    `orm:"column(cur_replicas)" json:"cur_replicas"`
	TemplateName      string `orm:"column(template_name)" json:"template_name"`
	Image             string `orm:"column(image)" json:"image"`
	Template          string `orm:"column(template);type(text)" json:"template,omitempty"`
	InjectServiceMesh string `orm:"column(inject_service_mesh)" json:"inject_service_mesh,omitempty"`
	Addons
}

const (
	STAGE_NEW         = "new"
	STAGE_NORMAL      = "normal"
	MIN_WEIGHT        = 0
	DEF_INIT_WEIGHT   = 10
	MAX_WEIGHT        = 100
	DEFAULT_WEIGHT    = 1
	INIT_CUR_REPLICAS = 1
)

func (t *ZcloudVersion) TableName() string {
	return "zcloud_version"
}
