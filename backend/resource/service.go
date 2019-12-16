package resource

import (
	"kubecloud/backend/dao"
	"kubecloud/common/utils"

	"fmt"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	apiv1 "k8s.io/api/core/v1"
)

type SimpleService struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ports     []int             `json:"ports"`
	Type      apiv1.ServiceType `json:"type"`
	ClusterIP string            `json:"clusterIP"`
}

type serviceAddress struct {
	Port         int    `json:"port"`
	TargetPort   int    `json:"target_port"`
	NodePort     int    `json:"node_port"`
	Protocol     string `json:"protocol"`
	NodePortAddr string `json:"node_port_addr"`
	ClusterAddr  string `json:"cluster_addr"`
}

type podSvcAddress struct {
	TargetPort  int    `json:"target_port"`
	Protocol    string `json:"protocol"`
	ClusterAddr string `json:"cluster_addr"`
}

type ServiceDetail struct {
	Name           string           `json:"name"`
	Type           string           `json:"type"`
	ClusterIP      string           `json:"cluster_ip"`
	AddressList    []serviceAddress `json:"address_list"`
	PodsvcAddrList []podSvcAddress  `json:"podsvc_addr_list"`
}

type ServiceRes struct {
	cluster    string
	modelSvc   *dao.K8sServiceModel
	listNSFunc NamespaceListFunction
}

func NewServiceRes(cluster string, get NamespaceListFunction) *ServiceRes {
	return &ServiceRes{cluster: cluster, modelSvc: dao.NewK8sServiceModel(), listNSFunc: get}
}

func (sr *ServiceRes) GetServiceList(namespace string) ([]SimpleService, error) {
	list := []SimpleService{}
	svcPortList, err := sr.modelSvc.List(sr.cluster, namespace, 0)
	if err != nil {
		beego.Debug("error:", err)
		return list, err
	}
	nslist := sr.listNSFunc()
	for _, port := range svcPortList {
		// filter headless service
		if !utils.ContainsString(nslist, port.Namespace) ||
			port.Service.ClusterIP == "None" {
			continue
		}
		index := -1
		for i, item := range list {
			if port.Service.Name == item.Name {
				index = i
				break
			}
		}
		if index >= 0 {
			// just append port
			list[index].Ports = append(list[index].Ports, port.Port)
		} else {
			svc := SimpleService{
				Name:      port.Service.Name,
				Namespace: port.Service.Namespace,
				Type:      apiv1.ServiceType(port.Service.Type),
				ClusterIP: port.Service.ClusterIP,
			}
			svc.Ports = append(svc.Ports, port.Port)
			list = append(list, svc)
		}
	}

	return list, nil
}

func (sr *ServiceRes) GetService(namespace, owner, name string) ([]SimpleService, error) {
	service, err := sr.modelSvc.Get(sr.cluster, namespace, owner, name)
	if err != nil {
		return nil, err
	}
	ss := SimpleService{
		Name:      service.Name,
		Namespace: service.Namespace,
		Type:      apiv1.ServiceType(service.Type),
		ClusterIP: service.ClusterIP,
	}
	for _, port := range service.Ports {
		ss.Ports = append(ss.Ports, port.Port)
	}

	return []SimpleService{ss}, nil
}

func (sr *ServiceRes) GetServiceDetail(namespace, name, nodeip string) (ServiceDetail, error) {
	svcDetail := ServiceDetail{}
	svc, err := sr.modelSvc.Get(sr.cluster, namespace, name, "")
	if err != nil {
		if err != orm.ErrNoRows {
			beego.Error("Get service information failed: " + err.Error())
			return svcDetail, err
		}
		return svcDetail, nil
	}
	svcDetail.Name = svc.OwnerName
	svcDetail.Type = svc.Type
	svcDetail.ClusterIP = svc.ClusterIP
	for _, item := range svc.Ports {
		var address serviceAddress
		address.Port = item.Port
		address.TargetPort = item.TargetPort
		address.Protocol = item.Protocol
		address.NodePort = item.NodePort
		address.ClusterAddr = fmt.Sprintf("%s.%s:%v", svc.Name, svc.Namespace, address.Port)
		if apiv1.ServiceType(svcDetail.Type) == apiv1.ServiceTypeNodePort && nodeip != "" {
			address.NodePortAddr = fmt.Sprintf("%s:%v", nodeip, address.NodePort)
		} else {
			address.NodePortAddr = "<none>"
		}
		svcDetail.AddressList = append(svcDetail.AddressList, address)
	}

	return svcDetail, nil
}

func (sr *ServiceRes) GetHeadlessSvcDetail(namespace, name string) (ServiceDetail, error) {
	svcDetail := ServiceDetail{}
	svc, err := sr.modelSvc.Get(sr.cluster, namespace, name, "")
	if err != nil {
		beego.Error("Get service information failed:", err)
		return svcDetail, nil
	}
	svcDetail.Name = svc.OwnerName
	svcDetail.Type = svc.Type
	svcDetail.ClusterIP = svc.ClusterIP
	for _, item := range svc.Ports {
		var address serviceAddress
		address.Port = item.Port
		address.TargetPort = item.TargetPort
		address.Protocol = item.Protocol
		address.ClusterAddr = fmt.Sprintf("%s.%s.svc.cluster.local:%v", svc.Name, svc.Namespace, address.Port)
		address.NodePortAddr = "<none>"
		svcDetail.AddressList = append(svcDetail.AddressList, address)
		ep, err := dao.NewK8sEndpointModel().Get(sr.cluster, namespace, svc.Name, int32(item.TargetPort))
		if err != nil {
			beego.Warn("get endpoint failed:", err)
		} else {
			for _, addr := range ep.Addresses {
				svcDetail.PodsvcAddrList = append(svcDetail.PodsvcAddrList, podSvcAddress{
					TargetPort:  item.TargetPort,
					Protocol:    item.Protocol,
					ClusterAddr: fmt.Sprintf("%s.%s.%s:%v", addr.TargetRefName, svc.Name, svc.Namespace, item.TargetPort),
				})
			}
		}
	}

	return svcDetail, nil
}
