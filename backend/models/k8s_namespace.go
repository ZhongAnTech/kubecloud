package models

type K8sNamespace struct {
	ID          int64  `orm:"pk;column(id);auto" json:"id"`
	Cluster     string `orm:"column(cluster);index" json:"cluster"`
	Name        string `orm:"column(name);index" json:"name"`
	Desc        string `orm:"column(desc)" json:"desc"`
	CPUQuota    string `orm:"column(cpu_quota)" json:"cpu_quota"`
	MemoryQuota string `orm:"column(memory_quota)" json:"memory_quota"`
	AddonsUnix
}

func (t *K8sNamespace) TableName() string {
	return "k8s_namespace"
}
