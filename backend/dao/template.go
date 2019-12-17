package dao

import (
	"fmt"
	models "kubecloud/backend/models"

	"kubecloud/common/utils"

	"github.com/astaxie/beego/orm"
)

type TemplateModel struct {
	tOrmer    orm.Ormer
	TableName string
}

var templateEnableFilterKeys = map[string]interface{}{
	"namespace":   nil,
	"name":        nil,
	"description": nil,
	"creator":     nil,
	"create_at":   nil,
}

func NewTemplateModel() *TemplateModel {
	return &TemplateModel{
		tOrmer:    GetOrmer(),
		TableName: (&models.ZcloudTemplate{}).TableName(),
	}
}

func (tm *TemplateModel) CreateTemplate(template models.ZcloudTemplate) (*models.ZcloudTemplate, error) {
	template.Addons = models.NewAddons()
	_, err := tm.tOrmer.Insert(&template)
	if err != nil {
		return nil, err
	}

	return tm.GetTemplate(template.Namespace, template.Name)
}

func (tm *TemplateModel) UpdateTemplate(template models.ZcloudTemplate) error {
	template.Addons = template.Addons.UpdateAddons()
	_, err := tm.tOrmer.Update(&template)

	return err
}

func (tm *TemplateModel) DeleteTemplate(namespace, name string) error {
	sql := "update " + tm.TableName + " set deleted=1, delete_at=now() where namespace=? and name=? and deleted=0"
	_, err := tm.tOrmer.Raw(sql, namespace, name).Exec()

	return err
}

func (tm *TemplateModel) GetTemplate(namespace, name string) (*models.ZcloudTemplate, error) {
	var template models.ZcloudTemplate

	if err := tm.tOrmer.QueryTable(tm.TableName).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("deleted", 0).One(&template); err != nil {
		return nil, err
	}

	return &template, nil
}

func (tm *TemplateModel) GetTemplateByID(templateId int64) (*models.ZcloudTemplate, error) {
	var template models.ZcloudTemplate

	if err := tm.tOrmer.QueryTable(tm.TableName).
		Filter("id", templateId).
		Filter("deleted", 0).One(&template); err != nil {
		return nil, err
	}

	return &template, nil
}

func (tm *TemplateModel) GetTemplateList(nslist []string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	tList := []models.ZcloudTemplate{}

	query := tm.tOrmer.QueryTable(tm.TableName).
		Filter("deleted", 0).OrderBy("-create_at")
	if len(nslist) >= 1 {
		query = query.Filter("namespace__in", nslist)
	} else {
		return nil, fmt.Errorf("namespace must be given!")
	}
	if filterQuery.FilterVal != "" {
		if _, exist := templateEnableFilterKeys[filterQuery.FilterKey]; exist {
			query = query.Filter(filterQuery.FilterKey+"__icontains", filterQuery.FilterVal)
		}
	}
	if filterQuery.PageSize != 0 {
		realIndex := 0
		if filterQuery.PageIndex > 0 {
			realIndex = filterQuery.PageIndex - 1
		}
		query = query.Limit(filterQuery.PageSize, filterQuery.PageSize*realIndex)
	}
	if _, err := query.All(&tList); err != nil {
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
		List: tList}, err
}
