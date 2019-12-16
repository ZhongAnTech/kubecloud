package endpoint

import (
	"kubecloud/backend/models"

	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	core "k8s.io/api/core/v1"
)

// update if the endpoint is existed, or add it
func (ec *EndpointController) syncEndpointRecord(endpoint core.Endpoints) error {
	if len(endpoint.Subsets) == 0 {
		return nil
	}
	for index, port := range endpoint.Subsets[0].Ports {
		old, err := ec.endpointHandler.Get(ec.cluster, endpoint.Namespace,
			endpoint.Name, port.Port)
		if err != nil {
			if err != orm.ErrNoRows {
				return err
			}
			err = ec.createEndpointRecord(endpoint, 0, index)
		} else {
			err = ec.updateEndpointRecord(endpoint, *old, 0, index)
		}
		if err != nil {
			return err
		}
	}
	// delete some ash port
	list, err := ec.endpointHandler.ListByName(ec.cluster, endpoint.Namespace, endpoint.Name)
	if err != nil {
		return err
	}
	for _, item := range list {
		found := false
		for _, port := range endpoint.Subsets[0].Ports {
			if item.Port == port.Port {
				found = true
				break
			}
		}
		if !found {
			ec.endpointHandler.DeleteByID(item.Id)
		}
	}
	return nil
}

func (ec *EndpointController) deleteEndpointRecord(namespace, name string) error {
	// delete service
	err := ec.endpointHandler.Delete(ec.cluster, namespace, name)
	if err != nil {
		beego.Error("Delete kube service record", ec.cluster, namespace, name, "failed for", err)
	}
	return err
}

func (ec *EndpointController) createEndpointRecord(endpoint core.Endpoints, subnetIndex, portIndex int) error {
	record := genEndpointRecord(ec.cluster, endpoint, subnetIndex, portIndex)
	if len(record.Addresses) == 0 {
		return fmt.Errorf("Create kube endpoint(%s/%s/%s) record failed: has no endpoint addresses!", ec.cluster, record.Namespace, record.Name)
	}
	err := ec.endpointHandler.Create(record)
	if err != nil {
		beego.Error("Create kube endpoint record", ec.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}

func (ec *EndpointController) updateEndpointRecord(endpoint core.Endpoints, old models.K8sEndpoint, subnetIndex, portIndex int) error {
	record := genEndpointRecord(ec.cluster, endpoint, subnetIndex, portIndex)
	err := ec.endpointHandler.Update(old, record)
	if err != nil {
		beego.Error("Update kube endpoint", ec.cluster, record.Namespace, record.Name, "failed for", err)
		return err
	}
	return nil
}
