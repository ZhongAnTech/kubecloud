package kubeutil

import (
	apiv1 "k8s.io/api/core/v1"
	"strconv"
)

func SetTrafficWeight(svc *apiv1.Service, versionKey string, weight int) {
	if svc == nil {
		return
	}
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	svc.Annotations[versionKey] = strconv.Itoa(weight)
}
