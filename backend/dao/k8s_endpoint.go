package dao

import (
	models "kubecloud/backend/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type K8sEndpointModel struct {
	tOrmer        orm.Ormer
	endpointTable string
	addressTable  string
}

func NewK8sEndpointModel() *K8sEndpointModel {
	return &K8sEndpointModel{
		tOrmer:        GetOrmer(),
		endpointTable: (&models.K8sEndpoint{}).TableName(),
		addressTable:  (&models.K8sEndpointAddress{}).TableName(),
	}
}

func (em *K8sEndpointModel) Create(endpoint models.K8sEndpoint) error {
	endpoint.Addons = models.NewAddons()
	_, err := em.tOrmer.Insert(&endpoint)
	if err != nil {
		return err
	}
	var addresses []models.K8sEndpointAddress
	for _, item := range endpoint.Addresses {
		addr := *item
		addr.Endpoint = &endpoint
		addr.Addons = models.NewAddons()
		addresses = append(addresses, addr)
	}
	_, err = em.tOrmer.InsertMulti(len(addresses), addresses)
	return err
}

func (em *K8sEndpointModel) Update(old, cur models.K8sEndpoint) error {
	cur.Id = old.Id
	if !em.endpointBaseIsEqual(old, cur) {
		cur.OwnerName = old.OwnerName
		if err := em.updateEndpoint(old, cur); err != nil {
			return err
		}
	}
	for _, na := range cur.Addresses {
		index := -1
		for i, oa := range old.Addresses {
			if em.addressIsEqual(*na, *oa) {
				index = i
			}
		}
		if index == -1 {
			// create
			if err := em.insertAddress(na, &cur); err != nil {
				return err
			}
		}
	}
	// delete old other
	for _, oa := range old.Addresses {
		index := -1
		for i, na := range cur.Addresses {
			if em.addressIsEqual(*oa, *na) {
				index = i
				break
			}
		}
		if index == -1 {
			err := em.deleteAddress(oa.Id)
			if err != nil {
				beego.Error(err)
			}
		}
	}

	return nil
}

func (em *K8sEndpointModel) Delete(cluster, namespace, name string) error {
	err := em.deleteEndpoint(cluster, namespace, name)
	if err == nil {
		err = em.deleteAddresses(cluster, namespace, name)
	}
	return err
}

func (em *K8sEndpointModel) DeleteByID(id int64) error {
	sql := "delete from " + em.endpointTable + " where id=?"
	_, err := em.tOrmer.Raw(sql, id).Exec()
	if err == nil {
		sql := "delete from " + em.addressTable + " where endpoint=?"
		_, err = em.tOrmer.Raw(sql, id).Exec()
	}
	return err
}

func (em *K8sEndpointModel) Get(cluster, namespace, name string, port int32) (*models.K8sEndpoint, error) {
	var obj models.K8sEndpoint

	if err := em.tOrmer.QueryTable(em.endpointTable).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("port", port).
		Filter("deleted", 0).One(&obj); err != nil {
		beego.Error(err)
		return nil, err
	}
	if _, err := em.tOrmer.LoadRelated(&obj, "addresses"); err != nil {
		beego.Error(err)
		return nil, err
	}
	var addresses []*models.K8sEndpointAddress
	for _, addr := range obj.Addresses {
		if addr.Deleted == 0 {
			addresses = append(addresses, addr)
		}
	}
	obj.Addresses = addresses
	return &obj, nil
}

func (em *K8sEndpointModel) ListByName(cluster, namespace, name string) ([]models.K8sEndpoint, error) {
	list := []models.K8sEndpoint{}
	var err error

	query := em.tOrmer.QueryTable(em.endpointTable).
		Filter("cluster", cluster).
		Filter("namespace", namespace).
		Filter("name", name).
		Filter("deleted", 0).OrderBy("-create_at")
	_, err = query.All(&list)
	if err != nil {
		return list, err
	}

	return list, nil
}

func (im *K8sEndpointModel) updateEndpoint(old, cur models.K8sEndpoint) error {
	cur.Addons = old.Addons.UpdateAddons()
	_, err := im.tOrmer.Update(&cur)
	return err
}

func (em *K8sEndpointModel) deleteEndpoint(cluster, namespace, name string) error {
	sql := "delete from " + em.endpointTable + " where cluster=? and namespace=? and name=?"
	_, err := em.tOrmer.Raw(sql, cluster, namespace, name).Exec()
	return err
}

func (im *K8sEndpointModel) insertAddress(addr *models.K8sEndpointAddress, endpoint *models.K8sEndpoint) error {
	addr.Addons = models.NewAddons()
	addr.Endpoint = endpoint
	_, err := im.tOrmer.Insert(addr)
	return err
}

func (em *K8sEndpointModel) deleteAddresses(cluster, namespace, name string) error {
	sql := "delete from " + em.addressTable + " where cluster=? and namespace=? and endpoint_name=?"
	_, err := em.tOrmer.Raw(sql, cluster, namespace, name).Exec()

	return err
}

func (em *K8sEndpointModel) deleteAddress(id int64) error {
	sql := "delete from " + em.addressTable + " where id=?"
	_, err := em.tOrmer.Raw(sql, id).Exec()

	return err
}

func (em *K8sEndpointModel) addressIsEqual(a1, a2 models.K8sEndpointAddress) bool {
	if a1.Cluster != a2.Cluster ||
		a1.Namespace != a2.Namespace ||
		a1.EndpointName != a2.EndpointName ||
		a1.IP != a2.IP ||
		a1.NodeName != a2.NodeName ||
		a1.TargetRefName != a2.TargetRefName {
		return false
	}
	return true
}

func (em *K8sEndpointModel) endpointBaseIsEqual(e1, e2 models.K8sEndpoint) bool {
	if e1.OwnerName != e2.OwnerName ||
		e1.Port != e2.Port ||
		e1.PortName != e2.PortName ||
		e1.Protocol != e2.Protocol {
		return false
	}

	return true
}
