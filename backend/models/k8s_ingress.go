package models

type K8sIngress struct {
	Id int64 `orm:"pk;column(id);auto"`
	// name is unique
	Name       string            `orm:"column(name)"`
	Namespace  string            `orm:"column(namespace)"`
	Cluster    string            `orm:"column(cluster)"`
	Rules      []*K8sIngressRule `orm:"reverse(many);column(rules)"`
	Annotation string            `orm:"column(annotation);type(text)"`
	Addons
}

func (t *K8sIngress) TableName() string {
	return "k8s_ingress"
}

func (u *K8sIngress) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "Name"},
	}
}
