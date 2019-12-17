package models

const (
	ClusterStatusToBeConfigured string = "ToBeConfigured"
	ClusterStatusPending        string = "Pending"
	ClusterStatusRunning        string = "Running"
	ClusterStatusError          string = "Error"
	ClusterStatusUpdating       string = "Updating"
)

type ZcloudCluster struct {
	Id                     int64  `orm:"pk;column(id);auto" json:"id"`
	Name                   string `orm:"column(name)" json:"name"`
	ClusterId              string `orm:"column(cluster_id)" json:"cluster_id"`
	Tenant                 string `orm:"column(tenant)" json:"tenant"`
	Env                    string `orm:"column(env)" json:"env"`
	Usage                  string `orm:"column(usage)" json:"usage"`
	DisplayName            string `orm:"column(display_name)" json:"display_name"`
	Registry               string `orm:"column(registry)" json:"registry"`
	RegistryName           string `orm:"column(registry_name)" json:"registry_name"`
	ImagePullAddr          string `orm:"column(image_pull_addr)" json:"image_pull_addr"`
	DockerVersion          string `orm:"column(docker_version)" json:"docker_version"`
	NetworkPlugin          string `orm:"column(network_plugin)" json:"network_plugin"`
	DomainSuffix           string `orm:"column(domain_suffix)" json:"domain_suffix"`
	Certificate            string `orm:"column(certificate);type(text)" json:"certificate"`
	PrometheusAddr         string `orm:"column(prometheus_addr)" json:"prometheus_addr"`
	Status                 string `orm:"column(status)" json:"status"`
	KubeVersion            string `orm:"column(kube_version)" json:"kube_version"`
	LoadbalancerDomainName string `orm:"column(lb_name)" json:"loadbalancer_domain_name"`
	LoadbalancerIP         string `orm:"column(lb_ip)" json:"loadbalancer_ip"`
	LoadbalancerPort       string `orm:"column(lb_port)" json:"loadbalancer_port"`
	KubeServiceAddress     string `orm:"column(kube_service_addr)" json:"kube_service_address"`
	KubePodSubnet          string `orm:"column(kube_pods_subnet)" json:"kube_pod_subnet"`
	TillerHost             string `orm:"column(tiller_host)" json:"tiller_host"`
	PromRuleIndex          int64  `orm:"column(prom_rule_index);default(0)" json:"prom_rule_index"`
	IngressSLB             string `orm:"column(ingress_slb)" json:"ingress_slb"`
	LabelPrefix            string `orm:"column(label_prefix)" json:"label_prefix"`
	ConfigRepo             string `orm:"column(config_repo)" json:"config_repo"`
	ConfigRepoBranch       string `orm:"column(config_repo_branch)" json:"config_repo_branch"`
	ConfigRepoToken        string `orm:"column(config_repo_token)" json:"config_repo_token"`
	LastCommitId           string `orm:"column(last_commit_id)" json:"last_commit_id"`
	Addons
}

func (t *ZcloudCluster) TableName() string {
	return "zcloud_cluster"
}

type ZcloudClusterDomainSuffix struct {
	Id           int64  `orm:"pk;column(id);auto" json:"id"`
	Cluster      string `orm:"column(cluster)" json:"cluster"`
	DomainSuffix string `orm:"column(domain_suffix)" json:"domain_suffix"`
	IsDefault    bool   `orm:"column(is_default)" json:"is_default"`
	Addons
}

func (t *ZcloudClusterDomainSuffix) TableName() string {
	return "zcloud_cluster_domain_suffix"
}
