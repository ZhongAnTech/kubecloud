package service

import (
	"kubecloud/backend/controllers/util"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"
	commutil "kubecloud/common/utils"

	core "k8s.io/api/core/v1"
)

//generate service template by service. return service struct
func genServiceRecord(cluster string, svc core.Service) models.K8sService {
	record := models.K8sService{
		Cluster:   cluster,
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}
	record.OwnerName = util.GetAnnotationStringValue(resource.OwnerNameAnnotationKey,
		svc.Annotations, resource.GetApplicationNameBySvcName(svc.Name))
	record.Type = string(svc.Spec.Type)
	record.ClusterIP = svc.Spec.ClusterIP
	record.Annotation = commutil.SimpleJsonMarshal(svc.Annotations, "")
	for _, item := range svc.Spec.Ports {
		port := models.K8sServicePort{
			Name:        item.Name,
			Cluster:     cluster,
			Namespace:   svc.Namespace,
			ServiceName: svc.Name,
			Protocol:    string(item.Protocol),
			Port:        int(item.Port),
			TargetPort:  item.TargetPort.IntValue(),
			NodePort:    int(item.NodePort),
		}
		record.Ports = append(record.Ports, &port)
	}

	return record
}

func serviceIsEqual(os, ns models.K8sService) bool {
	equal := true
	if os.ClusterIP != ns.ClusterIP ||
		os.Type != ns.Type ||
		len(os.Ports) != len(ns.Ports) ||
		os.Annotation != ns.Annotation {
		equal = false
	}
	for _, np := range ns.Ports {
		if !equal {
			break
		}
		exist := false
		for _, op := range os.Ports {
			if np.Name == op.Name {
				exist = true
				if np.Port != op.Port ||
					np.TargetPort != op.TargetPort ||
					np.Protocol != op.Protocol ||
					np.NodePort != op.NodePort {
					equal = false
				}
				break
			}
		}
		if !exist {
			equal = false
		}
	}
	return equal
}
