package models

type K8sIngressRule struct {
	Id          int64  `orm:"pk;column(id);auto" json:"id"`
	Cluster     string `orm:"column(cluster)" json:"cluster"`
	Namespace   string `orm:"column(namespace)" json:"namespace"`
	IngressName string `orm:"column(ingress_name)" json:"ingress_name,omitempty"`
	// host is unique
	Host        string      `orm:"column(host)" json:"host,omitempty"`
	Path        string      `orm:"column(path)" json:"path,omitempty"`
	SecretName  string      `orm:"column(secret_name)" json:"secret_name"`
	IsTls       bool        `orm:"column(is_tls)" json:"is_tls,omitempty"`
	ServiceName string      `orm:"column(service_name)" json:"service_name"`
	ServicePort int         `orm:"column(service_port)" json:"service_port"`
	Ingress     *K8sIngress `orm:"rel(fk);column(ingress)" json:"-"`
	Addons
}

func (t *K8sIngressRule) TableName() string {
	return "k8s_ingress_rule"
}

func (u *K8sIngressRule) TableUnique() [][]string {
	return [][]string{
		[]string{"Cluster", "Namespace", "IngressName"},
	}
}
