package dao

import (
	"fmt"

	"github.com/astaxie/beego/orm"

	"kubecloud/backend/models"
	"kubecloud/common/utils"
)

var nodeEnableFilterKeys = []string{
	"name",
	"department",
	"bizcluster",
	"status",
}

func CheckNodeExist(cluster, ip string) bool {
	ormer := GetOrmer()
	return ormer.QueryTable("zcloud_node").
		Filter("cluster", cluster).
		Filter("name", ip).Exist()
}

func CreateNode(node models.ZcloudNode) error {
	ormer := GetOrmer()
	if ormer.QueryTable("zcloud_node").
		Filter("cluster", node.Cluster).
		Filter("name", node.Name).Exist() {
		return fmt.Errorf("node(%s) is existed!", node.Name)
	}

	if _, err := ormer.Insert(&node); err != nil {
		return fmt.Errorf("database error, %s", err.Error())
	}
	return nil
}

func UpdateNode(node models.ZcloudNode) error {
	ormer := GetOrmer()
	if !ormer.QueryTable("zcloud_node").
		Filter("cluster", node.Cluster).
		Filter("name", node.Name).Exist() {
		return fmt.Errorf("node(%s) is not existed!", node.Name)
	}

	_, err := ormer.Update(&node)

	return err
}

func DeleteNode(cluster, name string) error {
	_, err := GetOrmer().Raw("delete from zcloud_node where cluster=? AND name=?", cluster, name).Exec()
	return err
}

func DeleteNodesByClusterName(clusterId string) error {
	_, err := GetOrmer().Raw("delete from zcloud_node where cluster=?", clusterId).Exec()
	return err
}

func GetNodeList(cluster string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	var nodes []*models.ZcloudNode
	queryCond := orm.NewCondition()
	if cluster != "" {
		queryCond = queryCond.And("cluster", cluster)
	}
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(nodeEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
	}

	query := GetOrmer().QueryTable("zcloud_node").OrderBy("-create_at").SetCond(queryCond)

	count, err := query.Count()
	if err != nil {
		return nil, err
	}

	if filterQuery != nil && filterQuery.PageSize != 0 && filterQuery.PageIndex > 0 {
		query = query.Limit(filterQuery.PageSize, filterQuery.PageSize*(filterQuery.PageIndex-1))
	}
	if _, err := query.All(&nodes); err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: filterQuery.PageIndex,
			PageSize:  filterQuery.PageSize,
		},
		List: nodes}, err
}
func GetNodeByName(cluster, name string) (*models.ZcloudNode, error) {
	var node models.ZcloudNode
	if err := GetOrmer().QueryTable("zcloud_node").
		Filter("cluster", cluster).
		Filter("name", name).One(&node); err != nil {
		if err == orm.ErrMultiRows {
			return nil, fmt.Errorf("node %s has multi rows", name)
		}
		return nil, err
	}

	return &node, nil
}

func GetNodeByIP(cluster, nodeIP string) (*models.ZcloudNode, error) {
	var node models.ZcloudNode
	if err := GetOrmer().QueryTable("zcloud_node").
		Filter("cluster", cluster).
		Filter("ip", nodeIP).One(&node); err != nil {
		if err == orm.ErrMultiRows {
			return nil, fmt.Errorf("node %s has multi rows", nodeIP)
		}
		if err == orm.ErrNoRows {
			return nil, fmt.Errorf("node %s no row found", nodeIP)
		}
		return nil, err
	}

	return &node, nil
}

func GetNodeListByStatus(cluster string, department string, status string) ([]*models.ZcloudNode, error) {
	qs := GetOrmer().QueryTable("zcloud_node")
	if cluster != "" {
		qs = qs.Filter("cluster", cluster)
	}
	if department != "all" {
		qs = qs.Filter("department__in", department)
	}
	if status != "all" {
		qs = qs.Filter("status", status)
	}
	var nodes []*models.ZcloudNode
	if _, err := qs.All(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}
