package dao

import (
	"fmt"
	"kubecloud/backend/models"
	"kubecloud/common/utils"

	"github.com/astaxie/beego/orm"
)

var eventEnableFilterKeys = []string{
	"event_type",
	"source_component",
	"source_host",
	"object_kind",
	"object_name",
	"reason",
	"message",
	"last_time",
}

func CreateEvent(event models.ZcloudEvent) error {
	_, err := GetOrmer().InsertOrUpdate(&event)
	return err
}

func GetEvents(clusterName, namespace, sourceHost, objectKind, objectName, eventLevel string, limitCount int64) ([]models.ZcloudEvent, error) {
	var events []models.ZcloudEvent
	qs := GetOrmer().QueryTable("zcloud_event").OrderBy("-last_time")
	if clusterName != "" {
		qs = qs.Filter("cluster", clusterName)
	}
	if namespace != "" {
		qs = qs.Filter("namespace", namespace)
	}
	if sourceHost != "" {
		qs = qs.Filter("source_host", sourceHost)
	}
	if eventLevel != "" {
		qs = qs.Filter("event_level", eventLevel)
	}

	if objectKind != "" {
		switch objectKind {
		case "Pod", "Node":
			qs = qs.Filter("object_kind", objectKind)
		default:
			err := fmt.Errorf("don't supported object kind: %s", objectKind)
			return events, err
		}
	}
	if objectName != "" {
		qs = qs.Filter("object_name", objectName)
	}

	if limitCount != -1 {
		qs = qs.Limit(limitCount)
	}
	if _, err := qs.All(&events); err != nil {
		return events, err
	}

	return events, nil
}

func GetAppEvents(cluster, namespace string, app string) ([]*models.ZcloudEvent, error) {
	events := []*models.ZcloudEvent{}
	sql := `select * from zcloud_event where cluster=? and namespace=? and object_name like '` + app + "%' order by last_time desc"
	if _, err := GetOrmer().Raw(sql, cluster, namespace).QueryRows(&events); err != nil {
		return nil, err
	}
	return events, nil
}

func GetNodeEvents(cluster, host string) ([]*models.ZcloudEvent, error) {
	events := []*models.ZcloudEvent{}
	ormer := GetOrmer()
	_, err := ormer.QueryTable("zcloud_event").
		Filter("cluster", cluster).
		Filter("object_kind", "Node").
		Filter("source_host", host).
		OrderBy("-last_time").All(&events)
	return events, err
}

func GetNodeEventsByFilter(cluster, host string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	var events []models.ZcloudEvent
	queryCond := orm.NewCondition().And("cluster", cluster).And("source_host", host)
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(eventEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
	}

	query := GetOrmer().QueryTable("zcloud_event").OrderBy("-last_time").SetCond(queryCond)

	count, err := query.Count()
	if err != nil {
		return nil, err
	}

	if filterQuery != nil && filterQuery.PageSize != 0 && filterQuery.PageIndex > 0 {
		query = query.Limit(filterQuery.PageSize, filterQuery.PageSize*(filterQuery.PageIndex-1))
	}
	if _, err := query.All(&events); err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: filterQuery.PageIndex,
			PageSize:  filterQuery.PageSize,
		},
		List: events}, err
}
