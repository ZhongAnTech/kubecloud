package models

type K8sEndpoint struct {
	Id        int64  `orm:"pk;column(id);auto"`
	Name      string `orm:"column(name)"`
	Cluster   string `orm:"column(cluster)"`
	Namespace string `orm:"column(namespace)"`
	OwnerName string `orm:"column(owner_name)"`
	// port is unique: port is service port's target_port
	Port      int32                 `orm:"column(port)"`
	PortName  string                `orm:"column(port_name)"`
	Protocol  string                `orm:"column(protocol)"`
	Addresses []*K8sEndpointAddress `orm:"reverse(many);column(addresses)"` //设置一对多关系
	Addons
}

func (t *K8sEndpoint) TableName() string {
	return "k8s_endpoint"
}

func (u *K8sEndpoint) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "Name"},
	}
}
