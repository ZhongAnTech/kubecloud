package secret

import (
	"kubecloud/backend/controllers/util"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"
	"time"

	"encoding/json"

	"github.com/astaxie/beego"
	core "k8s.io/api/core/v1"
)

//generate secret template by Secret. return K8sSecret struct
func genSecretRecord(cluster string, s core.Secret) models.K8sSecret {
	record := models.K8sSecret{
		Cluster:   cluster,
		Namespace: s.Namespace,
		Name:      s.Name,
		Addons:    models.NewAddons(),
	}
	record.CreateAt, _ = time.Parse("2006-01-02 15:04:05", s.CreationTimestamp.Local().Format("2006-01-02 15:04:05"))
	record.OwnerName = util.GetAnnotationStringValue(resource.OwnerNameAnnotationKey,
		s.Annotations, "")
	record.Description = util.GetAnnotationStringValue(resource.DescriptionAnnotationKey,
		s.Annotations, "")
	record.Type = string(s.Type)
	data, err := json.Marshal(&s.Data)
	if err != nil {
		beego.Error("marshal secret data failed:", s.Name, err)
	} else {
		record.Data = string(data)
	}

	return record
}

func secretIsEqual(os, ns models.K8sSecret) bool {
	equal := true
	if os.Type != ns.Type ||
		os.Description != ns.Description ||
		os.Data != ns.Data {
		equal = false
	}
	return equal
}
