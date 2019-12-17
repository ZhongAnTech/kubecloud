package dao

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	models "kubecloud/backend/models"
	"kubecloud/common/utils"
)

type IngressRuleModel struct {
	tOrmer    orm.Ormer
	TableName string
}

type hostOwner struct {
	cluster   string
	namespace string
}

const (
	SIMPLE_UNIQUE_MODE = "simple"
)

var ruleEnableFilterKeys = map[string]interface{}{
	"namespace":    nil,
	"ingress_name": nil,
	"secret_name":  nil,
	"service_name": nil,
	"host":         nil,
	"creator":      nil,
	"create_at":    nil,
}

func NewIngressRuleModel() *IngressRuleModel {
	return &IngressRuleModel{
		tOrmer:    GetOrmer(),
		TableName: (&models.K8sIngressRule{}).TableName(),
	}
}

func (im *IngressRuleModel) List(cluster string, nslist []string, svcname string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	rulelist := []models.K8sIngressRule{}
	query := im.tOrmer.QueryTable(im.TableName).
		Filter("deleted", 0).OrderBy("-create_at")
	if cluster != "" {
		query = query.Filter("cluster", cluster)
	}
	if len(nslist) >= 1 {
		query = query.Filter("namespace__in", nslist)
	}
	if svcname != "" {
		query = query.Filter("service_name", svcname)
	}
	if filterQuery.FilterVal != "" {
		if _, exist := ruleEnableFilterKeys[filterQuery.FilterKey]; exist {
			suffix := "__icontains"
			if !filterQuery.IsLike {
				suffix = ""
			}
			query = query.Filter(filterQuery.FilterKey+suffix, filterQuery.FilterVal)
		}
	}
	if filterQuery.PageSize != 0 {
		realIndex := 0
		if filterQuery.PageIndex > 0 {
			realIndex = filterQuery.PageIndex - 1
		}
		query = query.Limit(filterQuery.PageSize, filterQuery.PageSize*realIndex)
	}
	if _, err := query.RelatedSel().All(&rulelist); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: filterQuery.PageIndex,
			PageSize:  filterQuery.PageSize,
		},
		List: rulelist}, err
}

func (im *IngressRuleModel) GetByID(cluster, namespace string, id int64) (*models.K8sIngressRule, error) {
	var rule models.K8sIngressRule

	if err := im.tOrmer.QueryTable(im.TableName).
		Filter("id", id).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("deleted", 0).RelatedSel().One(&rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (im *IngressRuleModel) CheckHostUnique(cluster, namespace, host string) error {
	//check the host is existed in other cluster or other namespace
	filter := &utils.FilterQuery{}
	filter.FilterKey = "host"
	filter.FilterVal = host
	filterCluster := ""
	if beego.AppConfig.String("other::hostUniqueMode") == SIMPLE_UNIQUE_MODE {
		filterCluster = cluster
	}
	res, err := im.List(filterCluster, nil, "", filter)
	if err != nil {
		return err
	}
	ownerList := []hostOwner{}
	for _, item := range res.List.([]models.K8sIngressRule) {
		if item.Cluster != cluster || item.Namespace != namespace {
			ownerList = append(ownerList, hostOwner{cluster: item.Cluster, namespace: item.Namespace})
		}
	}
	if len(ownerList) > 0 {
		beego.Warn(fmt.Sprintf("domain name(%s) is existed in cluster(%v)!", host, ownerList))
		return fmt.Errorf("domain name(%s) is existed in cluster!", host)
	}
	return nil
}

//just check path is unique under host
func (im *IngressRuleModel) CheckPathsUniqueInHost(cluster, namespace, host string, paths []string, id int64) error {
	//check the host is existed in other cluster or other namespace
	filter := &utils.FilterQuery{}
	filter.FilterKey = "host"
	filter.FilterVal = host
	res, err := im.List(cluster, []string{namespace}, "", filter)
	if err != nil {
		return err
	}
	for _, path := range paths {
		for _, item := range res.List.([]models.K8sIngressRule) {
			if utils.PathsIsEqual(item.Path, path) {
				if (id > 0 && item.Id != id) || id < 0 {
					return fmt.Errorf("path(%s) is existed in domain(%s)!", utils.GetRootPath(path), host)
				}
			}
		}
	}
	return nil
}

//just check path is unique under host
func (im *IngressRuleModel) GetIngressNameByHost(cluster, namespace, host string) string {
	var rule models.K8sIngressRule

	if err := im.tOrmer.QueryTable(im.TableName).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("host", host).
		Filter("deleted", 0).RelatedSel().One(&rule); err != nil {
		return ""
	}
	return rule.IngressName
}
