package dao

import (
	"fmt"
	models "kubecloud/backend/models"
	"kubecloud/common"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type K8sServiceModel struct {
	tOrmer    orm.Ormer
	svcTable  string
	portTable string
}

func NewK8sServiceModel() *K8sServiceModel {
	return &K8sServiceModel{
		tOrmer:    GetOrmer(),
		svcTable:  (&models.K8sService{}).TableName(),
		portTable: (&models.K8sServicePort{}).TableName(),
	}
}

func (km *K8sServiceModel) Create(svc models.K8sService) error {
	svc.Addons = models.NewAddons()
	_, err := km.tOrmer.Insert(&svc)
	if err != nil {
		return err
	}
	var ports []models.K8sServicePort
	for _, item := range svc.Ports {
		port := *item
		port.Service = &svc
		port.Addons = models.NewAddons()
		ports = append(ports, port)
	}
	_, err = km.tOrmer.InsertMulti(len(ports), ports)
	return err
}

func (km *K8sServiceModel) Update(oldsvc, newsvc models.K8sService) error {
	if err := km.updateService(oldsvc, newsvc); err != nil {
		return err
	}
	for _, np := range newsvc.Ports {
		index := -1
		for i, op := range oldsvc.Ports {
			if np.Name == op.Name {
				index = i
				break
			}
		}
		if index >= 0 {
			// update
			if np.Port != oldsvc.Ports[index].Port ||
				np.TargetPort != oldsvc.Ports[index].TargetPort ||
				np.Protocol != oldsvc.Ports[index].Protocol ||
				np.NodePort != oldsvc.Ports[index].NodePort {
				if err := km.updatePort(oldsvc.Ports[index], np); err != nil {
					return err
				}
			}
		} else {
			// create
			if err := km.insertPort(np, &newsvc); err != nil {
				return err
			}
		}
	}
	// delete old other
	for _, op := range oldsvc.Ports {
		index := -1
		for i, np := range newsvc.Ports {
			if op.Name == np.Name {
				index = i
				break
			}
		}
		if index == -1 {
			err := km.deletePort(op.Id)
			if err != nil {
				beego.Error(err)
			}
		}
	}

	return nil
}

func (km *K8sServiceModel) Delete(cluster, namespace, name string) error {
	beego.Debug("delete service: %v", name)
	// delete svc
	if err := km.deleteService(cluster, namespace, name); err != nil {
		return err
	}
	// delete ports
	if err := km.deletePorts(cluster, namespace, name); err != nil {
		return err
	}
	return nil
}

func (km *K8sServiceModel) List(cluster, namespace string, nodePort int) ([]models.K8sServicePort, error) {
	var svcs []models.K8sServicePort

	query := km.tOrmer.QueryTable(km.portTable).
		Filter("cluster", cluster).
		Filter("deleted", 0)
	if namespace != common.AllNamespace {
		query = query.Filter("namespace", namespace)
	}
	if nodePort != 0 {
		query = query.Filter("node_port", nodePort)
	}
	if _, err := query.RelatedSel().All(&svcs); err != nil {
		return nil, err
	}

	return svcs, nil
}

func (km *K8sServiceModel) Get(cluster, namespace, oname, name string) (*models.K8sService, error) {
	var svc models.K8sService
	if oname == "" && name == "" {
		return nil, fmt.Errorf("service name or owner name must be given!")
	}
	query := km.tOrmer.QueryTable(km.svcTable).
		Filter("cluster", cluster).
		Filter("deleted", 0)
	if namespace != "" && namespace != common.AllNamespace {
		query = query.Filter("namespace", namespace)
	}
	if oname != "" {
		query = query.Filter("owner_name", oname)
	}
	if name != "" {
		query = query.Filter("name", name)
	}
	if err := query.One(&svc); err != nil {
		return nil, err
	}
	if _, err := km.tOrmer.LoadRelated(&svc, "ports"); err != nil {
		return nil, err
	}
	var ports []*models.K8sServicePort
	for _, port := range svc.Ports {
		if port.Deleted == 0 {
			ports = append(ports, port)
		}
	}
	svc.Ports = ports

	return &svc, nil
}

func (km *K8sServiceModel) updateService(oldsvc, newsvc models.K8sService) error {
	newsvc.Addons = oldsvc.Addons.UpdateAddons()
	_, err := km.tOrmer.Update(&newsvc)
	return err
}

func (km *K8sServiceModel) deleteService(cluster, namespace, name string) error {
	sql := "delete from " + km.svcTable + " where cluster=? and namespace=? and name=?"
	_, err := km.tOrmer.Raw(sql, cluster, namespace, name).Exec()
	return err
}

func (km *K8sServiceModel) insertPort(port *models.K8sServicePort, svc *models.K8sService) error {
	port.Service = svc
	port.Addons = models.NewAddons()
	_, err := km.tOrmer.Insert(port)
	return err
}

func (km *K8sServiceModel) updatePort(oldport, newport *models.K8sServicePort) error {
	newport.Id = oldport.Id
	newport.Addons = oldport.Addons.UpdateAddons()
	newport.Service = oldport.Service
	_, err := km.tOrmer.Update(newport)
	return err
}

func (km *K8sServiceModel) deletePorts(cluster, namespace, name string) error {
	sql := "delete from " + km.portTable + " where cluster=? and namespace=? and service_name=?"
	_, err := km.tOrmer.Raw(sql, cluster, namespace, name).Exec()
	return err
}

func (km *K8sServiceModel) deletePort(id int64) error {
	sql := "delete from " + km.portTable + " where id=?"
	_, err := km.tOrmer.Raw(sql, id).Exec()
	return err
}
