package service

import (
	"kubecloud/backend/controllers/util"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	core "k8s.io/api/core/v1"
)

// update if the app is existed, or add it
func (sc *ServiceController) syncServiceRecord(svc core.Service) error {
	old, err := sc.svcHandler.Get(sc.cluster, svc.Namespace,
		util.GetAnnotationStringValue(resource.OwnerNameAnnotationKey, svc.Annotations, resource.GetApplicationNameBySvcName(svc.Name)),
		svc.Name)
	if err != nil {
		if err != orm.ErrNoRows {
			return err
		}
		err = sc.createServiceRecord(svc)
	} else {
		err = sc.updateServiceRecord(svc, *old)
	}
	return err
}

func (sc *ServiceController) deleteServiceRecord(namespace, name string) error {
	// delete service
	err := sc.svcHandler.Delete(sc.cluster, namespace, name)
	if err != nil {
		beego.Error("Delete kube service record", sc.cluster, namespace, name, "failed for", err)
	}
	return err
}

func (sc *ServiceController) createServiceRecord(svc core.Service) error {
	record := genServiceRecord(sc.cluster, svc)
	err := sc.svcHandler.Create(record)
	if err != nil {
		beego.Error("Create kube service record", sc.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}

func (sc *ServiceController) updateServiceRecord(svc core.Service, old models.K8sService) error {
	record := genServiceRecord(sc.cluster, svc)
	if !serviceIsEqual(old, record) {
		record.Id = old.Id
	}
	if record.Id != 0 {
		err := sc.svcHandler.Update(old, record)
		if err != nil {
			beego.Error("Update kube service", sc.cluster, record.Namespace, record.Name, "failed for", err)
			return err
		}
	}
	return nil
}
