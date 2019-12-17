package models

type ZcloudTemplate struct {
	Id          int64  `orm:"pk;column(id);auto" json:"id"`
	Name        string `orm:"column(name)" json:"name"`
	Namespace   string `orm:"column(namespace)" json:"namespace"`
	Description string `orm:"column(description)" json:"description,omitempty"`
	Spec        string `orm:"column(spec);type(text)" json:"spec"` //TemplateSpec
	Kind        string `orm:"column(kind)" json:"kind"`
	Addons
}

func (t *ZcloudTemplate) TableName() string {
	return "zcloud_template"
}
