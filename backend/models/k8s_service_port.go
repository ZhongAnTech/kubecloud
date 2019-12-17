package models

type K8sServicePort struct {
	Id          int64       `orm:"pk;column(id);auto"`
	Name        string      `orm:"column(name)"`
	Cluster     string      `orm:"column(cluster)"`
	Namespace   string      `orm:"column(namespace)"`
	ServiceName string      `orm:"column(service_name)"`
	Protocol    string      `orm:"column(protocol)"`
	Port        int         `orm:"column(port)"`
	TargetPort  int         `orm:"column(target_port)"`
	NodePort    int         `orm:"column(node_port)"`
	Service     *K8sService `orm:"rel(fk);column(service)"` //设置一对多关系
	Addons
}

func (t *K8sServicePort) TableName() string {
	return "k8s_service_port"
}

func (u *K8sServicePort) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "ServiceName", "Port"},
	}
}
