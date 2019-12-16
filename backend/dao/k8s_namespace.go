package dao

import (
	"fmt"
	"kubecloud/common/utils"

	"kubecloud/backend/models"

	"github.com/astaxie/beego/orm"
)

var nsEnableFilterKeys = []string{
	"cluster",
	"name",
	"desc",
	"cpu_quota",
	"memory_quota",
}

type NamespaceModel struct {
	tOrmer    orm.Ormer
	TableName string
}

func NewNamespaceModel() *NamespaceModel {
	return &NamespaceModel{
		tOrmer:    GetOrmer(),
		TableName: (&models.K8sNamespace{}).TableName(),
	}
}

func NamespaceList(clusters []string, names []string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	var rows []models.K8sNamespace
	PageIndex := 0
	PageSize := 0
	realIndex := 0
	queryCond := orm.NewCondition().And("deleted", 0)
	defFilter := utils.NewDefaultFilter().AppendFilter("cluster", clusters, utils.FilterOperatorIn).
		AppendFilter("name", names, utils.FilterOperatorIn)
	normalCond := defFilter.DefaultFilterCondition()
	queryCond = queryCond.AndCond(normalCond)
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(nsEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
		if filterQuery.PageSize != 0 {
			if filterQuery.PageIndex > 0 {
				realIndex = filterQuery.PageIndex - 1
			}
		}
		PageIndex = filterQuery.PageIndex
		PageSize = filterQuery.PageSize
	}
	query := GetOrmer().QueryTable("k8s_namespace").OrderBy("-updated_at").SetCond(queryCond)
	if PageSize != 0 {
		query = query.Limit(PageSize, PageSize*realIndex)
	}
	if _, err := query.All(&rows); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: PageIndex,
			PageSize:  PageSize,
		},
		List: rows}, err
}

func GetClusterNamespaceList(cluster string) ([]string, error) {
	var nsList []string
	_, err := GetOrmer().QueryTable("k8s_namespace").Filter("cluster", cluster).All(&nsList, "name")
	return nsList, err
}

func NamespaceGet(cluster string, name string) (*models.K8sNamespace, error) {
	var row models.K8sNamespace
	err := GetOrmer().
		QueryTable("k8s_namespace").
		Filter("cluster", cluster).
		Filter("name", name).
		Filter("deleted", 0).
		One(&row)
	return &row, err
}

func NamespaceInsert(row *models.K8sNamespace) error {
	_, err := GetOrmer().Insert(row)
	return err
}

func NamespaceUpdate(row *models.K8sNamespace) error {
	_, err := GetOrmer().Update(row)
	return err
}

func NamespaceDelete(cluster, namespace string) error {
	ormer := GetOrmer()
	if ormer.QueryTable("zcloud_application").
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("deleted", 0).Exist() {
		return fmt.Errorf("can't delete a namesapce which still has running applications")
	}
	if ormer.QueryTable("zcloud_job").
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("deleted", 0).Exist() {
		return fmt.Errorf("can't delete a namesapce which still has running jobs")
	}
	row, err := NamespaceGet(cluster, namespace)
	if err != nil {
		return err
	}
	row.MarkDeleted()
	return NamespaceUpdate(row)
}

func NamespaceExists(cluster, namespace string) bool {
	return GetOrmer().
		QueryTable("k8s_namespace").
		Filter("cluster", cluster).
		Filter("name", namespace).
		Filter("deleted", 0).
		Exist()
}
