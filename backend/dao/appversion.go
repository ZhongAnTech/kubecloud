package dao

import (
	"fmt"

	"github.com/astaxie/beego/orm"
	"kubecloud/backend/models"
)

type VersionModel struct {
	tOrmer    orm.Ormer
	TableName string
}

func NewVersionModel() *VersionModel {
	return &VersionModel{
		tOrmer:    GetOrmer(),
		TableName: (&models.ZcloudVersion{}).TableName(),
	}
}

func (vm *VersionModel) GetVersionList(cluster, namespace, appname string) ([]models.ZcloudVersion, error) {
	list := []models.ZcloudVersion{}

	query := vm.tOrmer.QueryTable(vm.TableName).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", appname)
	_, err := query.All(&list)
	if err == orm.ErrNoRows {
		return list, nil
	}
	for i, item := range list {
		// compatible with old application
		if item.PodVersion == "" {
			list[i].PodVersion = item.Version
		}
	}
	return list, err
}

func (vm *VersionModel) GetVersion(cluster, namespace, appname, version string) (*models.ZcloudVersion, error) {
	v := models.ZcloudVersion{}

	query := vm.tOrmer.QueryTable(vm.TableName).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", appname).
		Filter("version", version)
	err := query.One(&v)
	if err != nil {
		return nil, err
	}
	// compatible with old application
	if v.PodVersion == "" {
		v.PodVersion = v.Version
	}
	return &v, nil
}

func (vm *VersionModel) SetVersionWeight(version *models.ZcloudVersion) error {
	if version == nil {
		return fmt.Errorf("version must be given!")
	}
	v, err := vm.GetVersion(version.Cluster, version.Namespace, version.Name, version.Version)
	if err == orm.ErrNoRows {
		// insert
		version.Addons = models.NewAddons()
		_, err = vm.tOrmer.Insert(version)
	} else {
		if err != nil {
			return err
		}
		v.Weight = version.Weight
		v.CurReplicas = version.CurReplicas
		v.Addons = v.Addons.UpdateAddons()
		_, err = vm.tOrmer.Update(v)
	}
	return err
}

func (vm *VersionModel) DeleteVersion(cluster, namespace, appname, version string) error {
	_, err := vm.tOrmer.Raw("DELETE FROM "+vm.TableName+" WHERE cluster=? AND namespace=? AND name=? AND version=?",
		cluster, namespace, appname, version).Exec()
	return err
}

func (vm *VersionModel) DeleteAllVersion(cluster, namespace, appname string) error {
	_, err := vm.tOrmer.Raw("DELETE FROM "+vm.TableName+" WHERE cluster=? AND namespace=? AND name=?",
		cluster, namespace, appname).Exec()
	return err
}
