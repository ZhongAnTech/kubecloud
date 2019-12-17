package dao

import (
	"fmt"
	"time"

	"kubecloud/backend/models"
	util "kubecloud/common/utils"

	"github.com/astaxie/beego/orm"
)

var amLocker *util.SyncLocker

type AppModel struct {
	tOrmer     orm.Ormer
	TableName  string
	AdminTable string
}

var appEnableFilterKeys = []string{
	"name",
	"namespace",
	"base_name",
	"domain_name",
	"template_name",
	"image",
	"creator",
	"create_at",
	"kind",
	"inject_service_mesh",
	"labels",
	"env",
}

func init() {
	if amLocker == nil {
		amLocker = util.NewSyncLocker()
	}
}

func NewAppModel() *AppModel {
	return &AppModel{
		tOrmer:    GetOrmer(),
		TableName: (&models.ZcloudApplication{}).TableName(),
	}
}

func (am *AppModel) GetAppList(defFilter *util.DefaultFilter, filterQuery *util.FilterQuery) (*util.QueryResult, error) {
	appList := []models.ZcloudApplication{}
	queryCond := orm.NewCondition().And("deleted", 0)
	PageIndex := 0
	PageSize := 0
	realIndex := 0
	if defFilter != nil {
		normalCond := defFilter.DefaultFilterCondition()
		queryCond = queryCond.AndCond(normalCond)
	}
	if filterQuery != nil {
		if filterQuery.FilterKey == "name" {
			if val, ok := filterQuery.FilterVal.(string); ok {
				filterQuery.FilterVal = util.GenerateStandardAppName(val)
			}
		}
		filterCond := filterQuery.FilterCondition(appEnableFilterKeys)
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
	query := am.tOrmer.QueryTable(am.TableName).OrderBy("-update_at").SetCond(queryCond)
	if PageSize != 0 {
		query = query.Limit(PageSize, PageSize*realIndex)
	}
	if _, err := query.All(&appList); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &util.QueryResult{
		Base: util.PageInfo{
			TotalNum:  count,
			PageIndex: PageIndex,
			PageSize:  PageSize,
		},
		List: appList}, err
}

func (am *AppModel) GetAppNameList(defFilter *util.DefaultFilter, filterQuery *util.FilterQuery) ([]string, error) {
	appList := []models.ZcloudApplication{}
	nameList := []string{}
	queryCond := orm.NewCondition().And("deleted", 0)
	if defFilter != nil {
		normalCond := defFilter.DefaultFilterCondition()
		queryCond = queryCond.AndCond(normalCond)
	}
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(appEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
	}
	query := am.tOrmer.QueryTable(am.TableName).SetCond(queryCond)
	if _, err := query.All(&appList, "name"); err != nil {
		return nil, err
	}
	for _, app := range appList {
		nameList = append(nameList, app.Name)
	}
	return nameList, nil
}

func (am *AppModel) GetAppByName(cluster, namespace, name string) (*models.ZcloudApplication, error) {
	var app models.ZcloudApplication
	if err := am.tOrmer.QueryTable(am.TableName).
		Filter("name", name).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("deleted", 0).One(&app); err != nil {
		return nil, err
	}
	return &app, nil
}

func (am *AppModel) GetImage(cluster, namespace, name string) (string, error) {
	var app models.ZcloudApplication
	err := am.tOrmer.QueryTable(am.TableName).
		Filter("name", name).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("deleted", 0).One(&app, "image")
	return app.Image, err
}

func (am *AppModel) InsertApp(ins models.ZcloudApplication) error {
	_, err := am.tOrmer.Insert(&ins)
	return err
}

func (am *AppModel) CreateApp(ins models.ZcloudApplication) error {
	if !ClusterIsExist(ins.Cluster) {
		return fmt.Errorf("cluster %s dose not exist in database", ins.Cluster)
	}
	if !NamespaceExists(ins.Cluster, ins.Namespace) {
		return fmt.Errorf("namespace %v dose not exist in database", ins.Namespace)
	}
	if am.AppExist(ins.Cluster, ins.Namespace, ins.Name) {
		return fmt.Errorf("app %s already exists in cluster %s", ins.Name, ins.Cluster)
	}
	ins.Addons = models.NewAddons()
	_, err := am.tOrmer.Insert(&ins)
	return err
}

func (am *AppModel) DeleteApp(app models.ZcloudApplication) error {
	_, err := am.tOrmer.Raw("UPDATE "+am.TableName+" SET deleted=1, delete_at=now() WHERE name=? AND cluster=? AND namespace=? AND deleted=0",
		app.Name, app.Cluster, app.Namespace).Exec()
	return err
}

func (am *AppModel) UpdateApp(ins *models.ZcloudApplication, updateTime bool) error {
	if ins == nil {
		return nil
	}
	amLocker.Lock(ins.Name)
	defer amLocker.Unlock(ins.Name)

	old, err := am.GetAppByName(ins.Cluster, ins.Namespace, ins.Name)
	if err != nil {
		return err
	}
	if old.UpdateAt != ins.UpdateAt {
		return fmt.Errorf("the application %s/%s/%s is updated by other routine!", ins.Cluster, ins.Namespace, ins.Name)
	}
	if updateTime {
		ins.Addons = ins.Addons.UpdateAddons()
	} else {
		ins.Addons = ins.Addons.FormatAddons()
	}
	_, err = am.tOrmer.Update(ins)

	return err
}

func (am *AppModel) SetLabels(cluster, namespace, name, labels string) error {
	_, err := am.tOrmer.Raw("UPDATE "+am.TableName+" SET labels=? WHERE cluster=? AND namespace=? AND name=? AND deleted=0",
		labels, cluster, namespace, name).Exec()
	return err
}

func (am *AppModel) SetDeployStatus(cluster, namespace, name, status string) error {
	_, err := am.tOrmer.Raw("UPDATE "+am.TableName+" SET deploy_status=? WHERE name=? AND cluster=? AND namespace=? AND deleted=0",
		status, name, cluster, namespace).Exec()
	return err
}

//return apps which namespace is different with gived namespace
func (am *AppModel) GetExoticAppListByName(cluster, namespace, name string) ([]models.ZcloudApplication, error) {
	var apps []models.ZcloudApplication
	_, err := am.tOrmer.QueryTable(am.TableName).
		Filter("name", name).
		Filter("cluster", cluster).
		Filter("deleted", 0).
		Exclude("namespace", namespace).All(&apps)
	return apps, err
}

func (am *AppModel) AppExist(cluster, namespace, name string) bool {
	return am.tOrmer.QueryTable(am.TableName).
		Filter("cluster", cluster).Filter("namespace", namespace).
		Filter("name", name).Filter("deleted", 0).Exist()
}

func GetAppList(startAt, endAt time.Time) ([]models.ZcloudApplication, error) {
	appList := []models.ZcloudApplication{}
	cond := orm.NewCondition()
	notDeleted := cond.And("deleted", 0)
	delete := cond.And("deleted", 1).And("DeleteAt__gt", startAt).And("DeleteAt__lte", endAt)
	condition := notDeleted.OrCond(delete)
	qs := GetOrmer().QueryTable("zcloud_application")
	qs = qs.SetCond(condition)
	_, err := qs.OrderBy("-create_at").All(&appList)
	return appList, err
}

func GetAllAppsByCluster(cluster string, nslist, apps []string) ([]models.ZcloudApplication, error) {
	appList := []models.ZcloudApplication{}
	querySeter := GetOrmer().QueryTable("zcloud_application").Filter("deleted", 0).OrderBy("-CreateAt")
	if cluster != "" {
		querySeter = querySeter.Filter("cluster", cluster)
	}
	if len(nslist) >= 1 {
		querySeter = querySeter.Filter("namespace__in", nslist)
	}
	if len(apps) >= 1 {
		querySeter = querySeter.Filter("name__in", apps)
	}
	_, err := querySeter.All(&appList)
	return appList, err
}

func CountAppByImage(image string) int64 {
	appList := []models.ZcloudApplication{}
	querySeter := GetOrmer().QueryTable("zcloud_application").Filter("deleted", 0).Filter("image", image)
	count, _ := querySeter.All(&appList)
	return count
}
