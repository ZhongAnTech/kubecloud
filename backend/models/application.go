package models

type ZcloudApplication struct {
	Id      int64  `orm:"pk;column(id);auto" json:"id"`
	Name    string `orm:"column(name)" json:"name"`
	Version string `orm:"column(version)" json:"version"`
	// pod_version is deployment version: deployment_name=app_name+pod_version, if pod_version is not empty
	// or version when pod_version is empty
	PodVersion        string `orm:"column(pod_version)" json:"pod_version"`
	BaseName          string `orm:"column(base_name)" json:"base_name"` //saas name
	Env               string `orm:"column(env)" json:"env"`
	DomainName        string `orm:"column(domain_name)" json:"domain_name"`
	Kind              string `orm:"column(kind)" json:"kind"`
	TemplateName      string `orm:"column(template_name)" json:"template_name"`
	Cluster           string `orm:"column(cluster)" json:"cluster"`
	Namespace         string `orm:"column(namespace)" json:"namespace"`
	Replicas          int    `orm:"column(replicas)" json:"replicas"`
	Image             string `orm:"column(image)" json:"image"`
	Template          string `orm:"column(template);type(text)" json:"template,omitempty"`
	LabelSelector     string `orm:"column(label_selector)" json:"label_selector,omitempty"`
	InjectServiceMesh string `orm:"column(inject_service_mesh)" json:"inject_service_mesh,omitempty"`
	StatusReplicas    int32  `orm:"column(status_replicas)" json:"status_replicas"`
	ReadyReplicas     int32  `orm:"column(ready_replicas)" json:"ready_replicas"`
	UpdatedReplicas   int32  `orm:"column(updated_replicas)" json:"updated_replicas"`
	AvailableReplicas int32  `orm:"column(available_replicas)" json:"available_replicas"`
	AvailableStatus   string `orm:"column(available_status)" json:"available_status"`
	Message           string `orm:"column(message)" json:"message"`
	DeployStatus      string `orm:"column(deploy_status)" json:"deploy_status"`
	DefaultDomainAddr string `orm:"column(default_domain_addr)" json:"default_domain_addr"`
	Labels            string `orm:"column(labels);type(text)" json:"labels"`
	Addons
}

func (t *ZcloudApplication) TableName() string {
	return "zcloud_application"
}
