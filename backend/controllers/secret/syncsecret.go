package secret

import (
	"kubecloud/backend/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	core "k8s.io/api/core/v1"
)

func (sc *SecretController) syncSecretRecord(s core.Secret) error {
	old, err := sc.secretHandler.GetSecret(sc.cluster, s.Namespace, s.Name)
	if err != nil {
		if err != orm.ErrNoRows {
			return err
		}
		err = sc.createSecretRecord(s)
	} else {
		err = sc.updateSecretRecord(s, *old)
	}

	return err
}

func (sc *SecretController) deleteSecretRecord(namespace, name string) error {
	err := sc.secretHandler.DeleteSecret(sc.cluster, namespace, name)
	if err != nil {
		beego.Error("Delete kube secret record", sc.cluster, namespace, name, "failed for", err)
	}
	return err
}

func (sc *SecretController) createSecretRecord(s core.Secret) error {
	record := genSecretRecord(sc.cluster, s)
	err := sc.secretHandler.CreateSecret(record, false)
	if err != nil {
		beego.Error("Create kube secret record", sc.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}

func (sc *SecretController) updateSecretRecord(s core.Secret, old models.K8sSecret) error {
	record := genSecretRecord(sc.cluster, s)
	if !secretIsEqual(old, record) {
		err := sc.secretHandler.UpdateSecret(old, record, false)
		if err != nil {
			beego.Error("Update kube secret", sc.cluster, record.Namespace, record.Name, "failed for", err)
			return err
		}
	}
	return nil
}
