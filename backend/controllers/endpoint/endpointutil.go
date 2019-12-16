package endpoint

import (
	"kubecloud/backend/controllers/util"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"

	core "k8s.io/api/core/v1"
)

//generate endpoint template by service. return endpoint struct
func genEndpointRecord(cluster string, endpoint core.Endpoints, subnetIndex, portIndex int) models.K8sEndpoint {
	record := models.K8sEndpoint{
		Cluster:   cluster,
		Namespace: endpoint.Namespace,
		Name:      endpoint.Name,
	}
	record.OwnerName = util.GetAnnotationStringValue(resource.OwnerNameAnnotationKey,
		endpoint.Annotations, resource.GetApplicationNameBySvcName(endpoint.Name))
	port := endpoint.Subsets[subnetIndex].Ports[portIndex]
	record.Port = port.Port
	record.PortName = port.Name
	record.Protocol = string(port.Protocol)
	// generate address
	for _, item := range endpoint.Subsets[subnetIndex].Addresses {
		address := models.K8sEndpointAddress{
			Cluster:      cluster,
			Namespace:    endpoint.Namespace,
			EndpointName: endpoint.Name,
			IP:           item.IP,
		}
		if item.NodeName != nil {
			address.NodeName = *item.NodeName
		}
		if item.TargetRef != nil {
			address.TargetRefName = item.TargetRef.Name
		}
		record.Addresses = append(record.Addresses, &address)
	}

	return record
}

func serviceIsEqual(os, ns models.K8sService) bool {
	equal := true
	if os.ClusterIP != ns.ClusterIP ||
		os.Type != ns.Type ||
		len(os.Ports) != len(ns.Ports) {
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
