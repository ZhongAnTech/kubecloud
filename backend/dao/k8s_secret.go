package dao

import (
	"fmt"

	"github.com/astaxie/beego/orm"

	"kubecloud/backend/models"
	"kubecloud/common/utils"
)

type SecretModel struct {
	ormer     orm.Ormer
	TableName string
}

var secretEnableFilterKeys = map[string]interface{}{
	"namespace":   nil,
	"name":        nil,
	"description": nil,
	"owner_kind":  nil,
	"owner_name":  nil,
	"type":        nil,
	"creator":     nil,
	"create_at":   nil,
}

func NewSecretModel() *SecretModel {
	return &SecretModel{
		ormer:     GetOrmer(),
		TableName: (&models.K8sSecret{}).TableName(),
	}
}

func (sm *SecretModel) Exists(cluster, namespace, name string) bool {
	return sm.ormer.QueryTable(sm.TableName).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("deleted", 0).
		Exist()
}

func (sm *SecretModel) CreateSecret(secret models.K8sSecret, setAddon bool) (err error) {
	if setAddon {
		secret.Addons = models.NewAddons()
	}
	_, err = sm.ormer.Insert(&secret)
	return
}

func (sm *SecretModel) UpdateSecret(old, cur models.K8sSecret, setAddon bool) (err error) {
	cur.Id = old.Id
	if setAddon {
		cur.Addons = old.Addons.UpdateAddons()
	}
	_, err = sm.ormer.Update(&cur)
	return
}

func (sm *SecretModel) DeleteSecret(cluster, namespace, name string) error {
	sql := "delete from " + sm.TableName + " where cluster=? and namespace=? and name=?"
	_, err := sm.ormer.Raw(sql, cluster, namespace, name).Exec()

	return err
}

func (sm *SecretModel) GetSecret(cluster, namespace, name string) (*models.K8sSecret, error) {
	var secret models.K8sSecret

	if err := sm.ormer.QueryTable(sm.TableName).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("deleted", 0).One(&secret); err != nil {
		return nil, err
	}

	return &secret, nil
}

func (sm *SecretModel) GetSecretList(cluster string, nslist []string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	secretList := []models.K8sSecret{}
	var err error
	query := sm.ormer.QueryTable(sm.TableName).
		Filter("cluster", cluster).
		Filter("deleted", 0).OrderBy("-create_at")
	if len(nslist) >= 1 {
		query = query.Filter("namespace__in", nslist)
	} else {
		return nil, fmt.Errorf("namespace must be given!")
	}
	if filterQuery.FilterVal != "" {
		if _, exist := secretEnableFilterKeys[filterQuery.FilterKey]; exist {
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
	if _, err := query.All(&secretList); err != nil {
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
		List: secretList}, err
}
