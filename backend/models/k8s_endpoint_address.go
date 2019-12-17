package models

type K8sEndpointAddress struct {
	Id            int64        `orm:"pk;column(id);auto"`
	Cluster       string       `orm:"column(cluster)"`
	Namespace     string       `orm:"column(namespace)"`
	EndpointName  string       `orm:"column(endpoint_name)"`
	IP            string       `orm:"column(ip)"`
	NodeName      string       `orm:"column(node_name)"`
	TargetRefName string       `orm:"column(target_ref_name)"`
	Endpoint      *K8sEndpoint `orm:"rel(fk);column(endpoint)"`
	Addons
}

func (t *K8sEndpointAddress) TableName() string {
	return "k8s_endpoint_address"
}
