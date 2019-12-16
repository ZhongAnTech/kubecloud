package models

type K8sSecret struct {
	Id          int64  `orm:"pk;column(id);auto" json:"id"`
	Name        string `orm:"column(name)" json:"name"`
	Cluster     string `orm:"column(cluster)" json:"cluster"`
	OwnerName   string `orm:"column(owner_name)"`
	Namespace   string `orm:"column(namespace)" json:"namespace"`
	Description string `orm:"column(description)" json:"description,omitempty"`
	Type        string `orm:"column(type)" json:"type,omitempty"`
	Data        string `orm:"column(data);type(text)" json:"data,omitempty"`
	Addons
}

func (t *K8sSecret) TableName() string {
	return "k8s_secret"
}

func (u *K8sSecret) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "Name"},
	}
}
