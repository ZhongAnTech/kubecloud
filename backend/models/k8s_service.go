package models

type K8sService struct {
	Id         int64             `orm:"pk;column(id);auto"`
	Name       string            `orm:"column(name)"`
	Cluster    string            `orm:"column(cluster)"`
	Namespace  string            `orm:"column(namespace)"`
	OwnerName  string            `orm:"column(owner_name)"`
	Type       string            `orm:"column(type)"`
	ClusterIP  string            `orm:"column(cluster_ip)"`
	Ports      []*K8sServicePort `orm:"reverse(many);column(ports)"`
	Annotation string            `orm:"column(annotation);type(text)"`
	Addons
}

func (t *K8sService) TableName() string {
	return "k8s_service"
}

func (u *K8sService) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "Name"},
	}
}
