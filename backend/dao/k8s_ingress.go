package dao

import (
	"fmt"

	"kubecloud/backend/models"
	"kubecloud/common"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type K8sIngressModel struct {
	tOrmer    orm.Ormer
	ingTable  string
	ruleTable string
}

func NewK8sIngressModel() *K8sIngressModel {
	return &K8sIngressModel{
		tOrmer:    GetOrmer(),
		ingTable:  (&models.K8sIngress{}).TableName(),
		ruleTable: (&models.K8sIngressRule{}).TableName(),
	}
}

func (im *K8sIngressModel) Create(ingress models.K8sIngress) error {
	ingress.Addons = models.NewAddons()
	_, err := im.tOrmer.Insert(&ingress)
	if err != nil {
		return err
	}
	var rules []models.K8sIngressRule
	for _, item := range ingress.Rules {
		rule := *item
		rule.Ingress = &ingress
		rule.Addons = models.NewAddons()
		rules = append(rules, rule)
	}
	_, err = im.tOrmer.InsertMulti(len(rules), rules)
	if err != nil {
		return err
	}
	return nil
}

func (im *K8sIngressModel) Update(oldIng, newIng models.K8sIngress) error {
	newIng.Id = oldIng.Id
	if !im.ingressBaseIsEqual(oldIng, newIng) {
		if err := im.updateIngress(oldIng, newIng); err != nil {
			return err
		}
	}
	for _, nr := range newIng.Rules {
		index := -1
		for i, or := range oldIng.Rules {
			if nr.Host == or.Host && utils.GetRootPath(nr.Path) == utils.GetRootPath(or.Path) {
				index = i
			}
		}
		if index >= 0 {
			// update
			if !im.ruleIsEqual(*nr, *oldIng.Rules[index]) {
				if err := im.updateRule(oldIng.Rules[index], nr); err != nil {
					return err
				}
			}
		} else {
			// create
			if err := im.insertRule(nr, &newIng); err != nil {
				return err
			}
		}
	}
	// delete old other
	for _, or := range oldIng.Rules {
		index := -1
		for i, nr := range newIng.Rules {
			if or.Host == nr.Host && utils.GetRootPath(or.Path) == utils.GetRootPath(nr.Path) {
				index = i
				break
			}
		}
		if index == -1 {
			im.deleteRules([]models.K8sIngressRule{*or})
		}
	}

	return nil
}

func (im *K8sIngressModel) Delete(cluster, namespace, name string) error {
	err := im.deleteIngress(cluster, namespace, name)
	if err == nil {
		rules, err := im.getRulesOfIngress(cluster, namespace, name)
		if err != nil {
			return err
		}
		im.deleteRules(rules)
	}
	return err
}

func (im *K8sIngressModel) Get(cluster, namespace, name string) (*models.K8sIngress, error) {
	var ingress models.K8sIngress

	if err := im.tOrmer.QueryTable(im.ingTable).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("deleted", 0).One(&ingress); err != nil {
		return nil, err
	}
	if _, err := im.tOrmer.LoadRelated(&ingress, "rules"); err != nil {
		return nil, err
	}
	// filter rule deleted = 0
	var rules []*models.K8sIngressRule
	for _, rule := range ingress.Rules {
		if rule.Deleted == 0 {
			rules = append(rules, rule)
		}
	}
	ingress.Rules = rules
	return &ingress, nil
}

func (im *K8sIngressModel) List(cluster, namespace string) ([]models.K8sIngress, error) {
	IngressList := []models.K8sIngress{}
	var err error

	query := im.tOrmer.QueryTable(im.ingTable).
		Filter("cluster", cluster).
		Filter("deleted", 0).OrderBy("-create_at")
	if namespace != common.AllNamespace {
		query = query.Filter("namespace", namespace)
	}
	_, err = query.All(&IngressList)
	if err != nil {
		return IngressList, err
	}

	return IngressList, nil
}

func (im *K8sIngressModel) DeleteRule(rule models.K8sIngressRule) error {
	im.deleteRules([]models.K8sIngressRule{rule})
	if len(rule.Ingress.Rules) == 1 {
		if rule.Ingress.Rules[0].Id == rule.Id {
			return im.deleteIngressByID(rule.Ingress.Id)
		}
	}
	return nil
}

func (im *K8sIngressModel) updateIngress(oldIng, newIng models.K8sIngress) error {
	newIng.Addons = oldIng.Addons.UpdateAddons()
	_, err := im.tOrmer.Update(&newIng)
	return err
}

func (im *K8sIngressModel) deleteIngress(cluster, namespace, name string) error {
	sql := "delete from " + im.ingTable + " where cluster=? and namespace=? and name=? and deleted=0"
	_, err := im.tOrmer.Raw(sql, cluster, namespace, name).Exec()
	return err
}

func (im *K8sIngressModel) deleteIngressByID(id int64) error {
	sql := "delete from " + im.ingTable + " where id=?"
	_, err := im.tOrmer.Raw(sql, id).Exec()
	return err
}

func (im *K8sIngressModel) insertRule(rule *models.K8sIngressRule, ing *models.K8sIngress) error {
	rule.Addons = models.NewAddons()
	rule.Ingress = ing
	_, err := im.tOrmer.Insert(rule)
	return err
}

func (im *K8sIngressModel) updateRule(oldRule, newRule *models.K8sIngressRule) error {
	newRule.Id = oldRule.Id
	newRule.Addons = oldRule.Addons.UpdateAddons()
	newRule.Ingress = oldRule.Ingress
	_, err := im.tOrmer.Update(newRule)
	return err
}

func (im *K8sIngressModel) getRulesOfIngress(cluster, namespace, name string) ([]models.K8sIngressRule, error) {
	rules := []models.K8sIngressRule{}
	_, err := im.tOrmer.QueryTable(im.ruleTable).Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("ingress_name", name).All(&rules)

	return rules, err
}

func (im *K8sIngressModel) deleteRules(rules []models.K8sIngressRule) {
	for _, rule := range rules {
		if rule.Id == 0 {
			continue
		}
		sql := "delete from " + im.ruleTable + " where id=?"
		if _, err := im.tOrmer.Raw(sql, rule.Id).Exec(); err != nil {
			beego.Warn(fmt.Sprintf("delete rule %v failed:", rule.Id, err))
		}
	}
}

func (im *K8sIngressModel) ruleIsEqual(i1, i2 models.K8sIngressRule) bool {
	if i1.Path != i2.Path ||
		i1.Host != i2.Host ||
		i1.IsTls != i2.IsTls ||
		i1.ServicePort != i2.ServicePort ||
		i1.ServiceName != i2.ServiceName ||
		i1.SecretName != i2.SecretName {
		return false
	}
	return true
}

func (im *K8sIngressModel) ingressBaseIsEqual(i1, i2 models.K8sIngress) bool {
	return i1.Annotation == i2.Annotation
}

func (im *K8sIngressModel) ListIngressID(cluster string, namespace []string) ([]models.K8sIngress, error) {
	IngressList := []models.K8sIngress{}
	var err error

	query := im.tOrmer.QueryTable(im.ingTable).
		Filter("cluster", cluster).
		Filter("deleted", 0).OrderBy("-create_at")

	if len(namespace) > 0 {
		query = query.Filter("namespace__in", namespace)
	}
	_, err = query.All(&IngressList, "id")
	if err != nil {
		return IngressList, err
	}

	return IngressList, nil
}

func (im *K8sIngressModel) GetIngressByID(cluster string, ids []string) ([]models.K8sIngressRule, error) {
	IngressList := []models.K8sIngressRule{}

	query := im.tOrmer.QueryTable(im.ruleTable).Filter("cluster", cluster).Filter("ingress__in", ids).Filter("deleted", 0).GroupBy("host", "path")

	_, err := query.All(&IngressList)
	if err != nil {
		return IngressList, err
	}

	return IngressList, nil
}
