package dao

import (
	"fmt"

	"github.com/astaxie/beego/orm"

	"kubecloud/backend/models"
	"kubecloud/common/utils"
)

var clusterEnableFilterKeys = []string{
	"name",
	"display_name",
	"env",
	"registry_name",
	"status",
	"domain_suffix",
	"creator",
	"create_at",
}

func GetClusterListByFilter(filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	var clusters []models.ZcloudCluster
	queryCond := orm.NewCondition().And("deleted", 0)
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(clusterEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
	}

	query := GetOrmer().QueryTable("zcloud_cluster").OrderBy("-create_at").SetCond(queryCond)

	count, err := query.Count()
	if err != nil {
		return nil, err
	}

	if filterQuery != nil && filterQuery.PageSize != 0 && filterQuery.PageIndex > 0 {
		query = query.Limit(filterQuery.PageSize, filterQuery.PageSize*(filterQuery.PageIndex-1))
	}
	if _, err := query.All(&clusters); err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: filterQuery.PageIndex,
			PageSize:  filterQuery.PageSize,
		},
		List: clusters}, err
}

func GetClusterList() ([]models.ZcloudCluster, error) {
	var clusters []models.ZcloudCluster
	qs := GetOrmer().QueryTable("zcloud_cluster").Filter("deleted", 0).OrderBy("-create_at")
	_, err := qs.All(&clusters)
	return clusters, err
}

func GetAllClusters() ([]models.ZcloudCluster, error) {
	return GetClusterList()
}

func GetClusterByTenant(tenant, name string) (*models.ZcloudCluster, error) {
	var cluster models.ZcloudCluster
	if err := GetOrmer().QueryTable("zcloud_cluster").
		Filter("tenant", tenant).Filter("name", name).One(&cluster); err != nil {
		return nil, err
	}
	return &cluster, nil
}

func GetCluster(clusterId string) (*models.ZcloudCluster, error) {
	var cluster models.ZcloudCluster
	if err := GetOrmer().QueryTable("zcloud_cluster").
		Filter("cluster_id", clusterId).One(&cluster); err != nil {
		if err == orm.ErrMultiRows {
			return nil, err
		}
		if err == orm.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("database error: get cluster(%s) info failed: %s!", clusterId, err.Error())
	}

	return &cluster, nil
}

func CreateCluster(cluster models.ZcloudCluster) error {
	if _, err := GetOrmer().Insert(&cluster); err != nil {
		return fmt.Errorf("database error: insert culster info failed: %s", cluster.Name)
	}
	return nil
}

func UpdateCluster(cluster models.ZcloudCluster) error {
	if !ClusterNameExist(cluster.ClusterId, cluster.Name) {
		return fmt.Errorf("database error: cluster(%s) is not existed!", cluster.Name)
	}

	if !HarborIsExist(cluster.Tenant, cluster.Registry) {
		return fmt.Errorf("default registry(%s) is not existed!", cluster.Registry)
	}

	_, err := GetOrmer().Update(&cluster)

	return err
}

func DeleteCluster(clusterId string) error {
	_, err := GetOrmer().Raw("delete from zcloud_cluster WHERE cluster_id=?", clusterId).Exec()
	return err
}

func ClusterIsExist(clusterId string) bool {
	return GetOrmer().QueryTable("zcloud_cluster").Filter("cluster_id", clusterId).Exist()
}

func ClusterNameExist(id, name string) bool {
	return GetOrmer().QueryTable("zcloud_cluster").Filter("cluster_id", id).Filter("name", name).Exist()
}

func IsClusterParamExistInTenant(tenant, fieldName, fieldValue string) bool {
	return GetOrmer().QueryTable("zcloud_cluster").Filter("tenant", tenant).Filter(fieldName, fieldValue).Exist()
}

func UpdateClusterStatus(clusterId, status string) error {
	_, err := GetOrmer().Raw("update zcloud_cluster set status = ? where cluster_id = ?", status, clusterId).Exec()
	return err
}

func GetClusterDomainSuffixList(clusterId string) ([]*models.ZcloudClusterDomainSuffix, error) {
	domainSuffixList := []*models.ZcloudClusterDomainSuffix{}
	_, err := GetOrmer().QueryTable("zcloud_cluster_domain_suffix").Filter("cluster", clusterId).OrderBy("-create_at").All(&domainSuffixList)
	return domainSuffixList, err
}

func AddClusterDomainSuffix(domainSuffix *models.ZcloudClusterDomainSuffix) error {
	_, err := GetOrmer().Insert(domainSuffix)
	return err
}

func DeleteClusterDomainSuffix(clusterId string) error {
	_, err := GetOrmer().Raw("delete from zcloud_cluster_domain_suffix where cluster=?", clusterId).Exec()
	return err
}
