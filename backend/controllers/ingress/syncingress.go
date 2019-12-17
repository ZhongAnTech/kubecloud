package ingress

import (
	"kubecloud/backend/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	extensions "k8s.io/api/extensions/v1beta1"
)

// update if the app is existed, or add it
func (ic *IngressController) syncIngressRecord(ing extensions.Ingress) error {
	old, err := ic.kubeIngHandler.Get(ic.cluster, ing.Namespace, ing.Name)
	if err != nil {
		if err != orm.ErrNoRows {
			return err
		}
		err = ic.createIngressRecord(ing)
	} else {
		err = ic.updateIngressRecord(ing, *old)
	}

	return err
}

func (ic *IngressController) deleteIngressRecord(namespace, name string) error {
	// delete ingress
	err := ic.kubeIngHandler.Delete(ic.cluster, namespace, name)
	if err != nil {
		beego.Error("Delete kube ingress record", ic.cluster, namespace, name, "failed for", err)
	}
	return err
}

func (ic *IngressController) createIngressRecord(ing extensions.Ingress) error {
	record := genIngressRecord(ic.cluster, ing)
	if len(record.Rules) == 0 {
		beego.Warn("no rules of ingress", ic.cluster, record.Namespace, record.Name)
		return nil
	}
	err := ic.kubeIngHandler.Create(record)
	if err != nil {
		beego.Error("Create kube ingress record", ic.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}

func (ic *IngressController) updateIngressRecord(ing extensions.Ingress, old models.K8sIngress) error {
	record := genIngressRecord(ic.cluster, ing)
	err := ic.kubeIngHandler.Update(old, record)
	if err != nil {
		beego.Error("Update kube ingress", ic.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}
