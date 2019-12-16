package resource

import (
	"fmt"
	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/util/kubeutil"
	"kubecloud/common/keyword"
	"kubecloud/gitops"

	"k8s.io/kubernetes/pkg/util/labels"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeAppRes struct {
	DomainSuffix  string
	Namespace     string
	cluster       string
	client        kubernetes.Interface
	kubeAppHandle KubeAppInterface
	gitOpsResList []interface{}
}

type RollBackFunc func() error

func NewKubeAppRes(client kubernetes.Interface, cluster, namespace, domainSuffix, kind string) *KubeAppRes {
	res := &KubeAppRes{
		DomainSuffix: domainSuffix,
		cluster:      cluster,
		Namespace:    namespace,
		client:       client,
	}
	if kind == AppKindDeployment {
		//default is deployment
		res.kubeAppHandle = NewDeploymentRes(client, cluster, namespace)
	}
	return res
}

func (kr *KubeAppRes) CreateAppResource(template AppTemplate, podVersion string) error {
	// rollback if create resource failed
	rollbackFuncList := []RollBackFunc{}
	defer func() {
		for _, rollback := range rollbackFuncList {
			if err := rollback(); err != nil {
				beego.Error(err)
			}
		}
	}()
	objMap, err := template.GenerateKubeObject(kr.cluster, kr.Namespace, podVersion, kr.DomainSuffix)
	if err != nil {
		return err
	}
	// create svc
	objs, existed := objMap[ServiceKind]
	if existed {
		svcList, ok := objs.([]*apiv1.Service)
		if !ok {
			return fmt.Errorf("service object list is not right")
		}
		// create or update
		if err := kr.CreateService(svcList, false); err != nil {
			return err
		}
		rollbackFuncList = append(rollbackFuncList, func() error { return kr.deleteService(svcList) })
	}
	// create ing
	obj, existed := objMap[IngressKind]
	if existed {
		ings, ok := obj.([]*extensions.Ingress)
		if !ok {
			return fmt.Errorf("ingress object list is not right")
		}
		// create or update
		if err := kr.CreateIngress(ings, template, false); err != nil {
			return err
		}
		rollbackFuncList = append(rollbackFuncList, func() error { return kr.deleteIngress(ings) })
	}
	// create app
	if obj, existed := objMap[template.GetAppKind()]; existed {
		if err := NewKubeAppValidator(kr.client, kr.Namespace).Validator(obj); err != nil {
			return err
		}
		// create or update
		app, err := kr.kubeAppHandle.CreateOrUpdate(obj)
		if err != nil {
			return err
		}
		kr.gitOpsResList = append(kr.gitOpsResList, app)
	}
	go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	rollbackFuncList = nil
	return nil
}

func (kr *KubeAppRes) UpdateAppResource(app *models.ZcloudApplication, new, old AppTemplate, all bool) error {
	err := new.UpdateAppObject(app, kr.DomainSuffix)
	if err != nil {
		return err
	}
	objMap, err := new.GenerateKubeObject(app.Cluster, app.Namespace, app.PodVersion, kr.DomainSuffix)
	if err != nil {
		return err
	}
	oldMap := make(map[string]interface{})
	if old != nil {
		oldMap, err = old.GenerateKubeObject(app.Cluster, app.Namespace, app.PodVersion, kr.DomainSuffix)
		if err != nil {
			return err
		}
	}
	if obj, existed := objMap[new.GetAppKind()]; existed {
		if err := NewKubeAppValidator(kr.client, kr.Namespace).Validator(obj); err != nil {
			return err
		}
		app, err := kr.kubeAppHandle.CreateOrUpdate(obj)
		if err != nil {
			return err
		}
		kr.gitOpsResList = append(kr.gitOpsResList, app)
	}
	if !all {
		go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
		return nil
	}
	if err := kr.updateSvcResource(oldMap[ServiceKind], objMap[ServiceKind]); err != nil {
		beego.Warn("update service resource failed:", err)
	}
	if err := kr.updateIngResource(oldMap[IngressKind], objMap[IngressKind], new); err != nil {
		beego.Warn("update ingress resource failed:", err)
	}
	go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	return nil
}

func (kr *KubeAppRes) DeleteAppResource(template AppTemplate, podVersion string) error {
	objMap, err := template.GenerateKubeObject(kr.cluster, kr.Namespace, podVersion, kr.DomainSuffix)
	if err != nil && objMap == nil {
		return err
	}
	if obj, existed := objMap[ServiceKind]; existed {
		svcList, ok := obj.([]*apiv1.Service)
		if !ok {
			return fmt.Errorf("service object list is not right")
		}
		// delete
		if err := kr.deleteService(svcList); err != nil {
			return fmt.Errorf("delete service error %v", err)
		}
	}

	if obj, existed := objMap[IngressKind]; existed {
		ings, ok := obj.([]*extensions.Ingress)
		if !ok {
			return fmt.Errorf("ingress object list is not right")
		}
		// delete
		if err := kr.deleteIngress(ings); err != nil {
			return err
		}
	}
	if obj, existed := objMap[template.GetAppKind()]; existed {
		app, err := kr.kubeAppHandle.Delete(obj)
		if err != nil {
			return err
		}
		kr.gitOpsResList = append(kr.gitOpsResList, app)
	}
	go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	return nil
}

func (kr *KubeAppRes) updateSvcResource(oldObjList, newObjList interface{}) error {
	delSvcs := []*apiv1.Service{}
	newSvcs := []*apiv1.Service{}
	ok := false
	if newObjList != nil {
		newSvcs, ok = newObjList.([]*apiv1.Service)
		if !ok {
			return fmt.Errorf("service object list is not right")
		}
		// create or update
		if err := kr.CreateService(newSvcs, false); err != nil {
			return err
		}
	}
	if oldObjList != nil {
		oldSvcList, _ := oldObjList.([]*apiv1.Service)
		for _, old := range oldSvcList {
			found := false
			for _, new := range newSvcs {
				if old.Name == new.Name {
					found = true
					break
				}
			}
			if !found {
				delSvcs = append(delSvcs, old)
			}
		}
	}
	// delete svc if only the old app has it
	if err := kr.deleteService(delSvcs); err != nil {
		beego.Warn("delete old service error:%v", err)
	}
	return nil
}

func (kr *KubeAppRes) updateIngResource(oldObjList, newObjList interface{}, newTpl AppTemplate) error {
	delIngs := []*extensions.Ingress{}
	newIngs := []*extensions.Ingress{}
	ok := false
	if newObjList != nil {
		newIngs, ok = newObjList.([]*extensions.Ingress)
		if !ok {
			return fmt.Errorf("ingress object list is not right")
		}
		// create or update
		if err := kr.CreateIngress(newIngs, newTpl, false); err != nil {
			return err
		}
	}
	if oldObjList != nil {
		oldIngs, _ := oldObjList.([]*extensions.Ingress)
		for _, old := range oldIngs {
			found := false
			for _, new := range newIngs {
				if old.Name == new.Name {
					found = true
					break
				}
			}
			if found {
				continue
			}
			ing, _ := kr.client.ExtensionsV1beta1().Ingresses(kr.Namespace).Get(old.Name, metav1.GetOptions{})
			if kubeutil.IngressIsCreatedDefault(ing) {
				delIngs = append(delIngs, old)
			}
		}
	}
	// delete ingress if only the old app default create them
	if err := kr.deleteIngress(delIngs); err != nil {
		beego.Warn("delete old ingress error:%v", err)
	}
	return nil
}

func (kr *KubeAppRes) SetAppStatus(app *models.ZcloudApplication) {
	status, err := kr.kubeAppHandle.Status(app.Name, app.PodVersion)
	if err != nil {
		beego.Error("get application status failed:", err)
		return
	}
	app.ReadyReplicas = status.ReadyReplicas
	app.UpdatedReplicas = status.UpdatedReplicas
	app.AvailableReplicas = status.AvailableReplicas
	app.AvailableStatus = status.AvailableStatus
	app.Message = status.Message
}

func (kr *KubeAppRes) Restart(app *models.ZcloudApplication, template AppTemplate) error {
	objMap, err := template.GenerateKubeObject(app.Cluster, app.Namespace, app.PodVersion, kr.DomainSuffix)
	if err != nil {
		return err
	}
	obj, ok := objMap[template.GetAppKind()]
	if ok {
		if err := NewKubeAppValidator(kr.client, kr.Namespace).Validator(obj); err != nil {
			return err
		}
	}
	return kr.kubeAppHandle.Restart(obj)
}

func (kr *KubeAppRes) DeleteApplication(app *models.ZcloudApplication, template AppTemplate) error {
	objMap, err := template.GenerateKubeObject(app.Cluster, app.Namespace, app.PodVersion, kr.DomainSuffix)
	if err != nil {
		return err
	}
	obj, ok := objMap[template.GetAppKind()]
	if ok {
		if err := NewKubeAppValidator(kr.client, kr.Namespace).Validator(obj); err != nil {
			return err
		}
	}
	res, err := kr.kubeAppHandle.Delete(obj)
	if err != nil {
		return err
	}
	kr.gitOpsResList = append(kr.gitOpsResList, res)
	go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	return nil
}

func (kr *KubeAppRes) CheckAppIsExisted(appname, suffix string) (bool, error) {
	return kr.kubeAppHandle.AppIsExisted(appname, suffix)
}

func (kr *KubeAppRes) Scale(app *models.ZcloudApplication, template AppTemplate, replicas int) error {
	objMap, err := template.GenerateKubeObject(app.Cluster, app.Namespace, app.PodVersion, kr.DomainSuffix)
	if err != nil {
		return err
	}
	obj, ok := objMap[template.GetAppKind()]
	if ok {
		if err := NewKubeAppValidator(kr.client, kr.Namespace).Validator(obj); err != nil {
			return err
		}
	}
	return kr.kubeAppHandle.Scale(obj, replicas)
}

func (kr *KubeAppRes) UpdateTrafficWeight(vs []models.ZcloudVersion) error {
	if len(vs) == 0 {
		return fmt.Errorf("versions must be given!")
	}
	service, err := dao.NewK8sServiceModel().Get(vs[0].Cluster, vs[0].Namespace, vs[0].Name, "")
	if err != nil {
		if err == orm.ErrNoRows {
			return nil
		}
		return err
	}
	svc, err := kr.client.CoreV1().Services(kr.Namespace).Get(service.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, vw := range vs {
		// use podVersion replace version, because traefik will get pod version from pod name.
		if vw.Weight == models.MIN_WEIGHT || vw.Weight == models.MAX_WEIGHT {
			delete(svc.Annotations, IngressWeightAnnotationKeyPre+vw.PodVersion)
		} else {
			kubeutil.SetTrafficWeight(svc, IngressWeightAnnotationKeyPre+vw.PodVersion, vw.Weight)
		}
	}
	_, err = kr.client.CoreV1().Services(svc.Namespace).Update(svc)
	return err
}

func (kr *KubeAppRes) CreateService(svcList []*apiv1.Service, commit bool) error {
	for _, svc := range svcList {
		//svcIsExisted := false
		old, err := kr.client.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			//svcIsExisted = true
			svc.ResourceVersion = old.ResourceVersion
			svc.Spec.ClusterIP = old.Spec.ClusterIP
			for i, port1 := range svc.Spec.Ports {
				for _, port2 := range old.Spec.Ports {
					if port1.TargetPort != port2.TargetPort {
						continue
					}
					svc.Spec.Ports[i].Name = port2.Name
					if svc.Spec.Type == apiv1.ServiceTypeNodePort && port1.NodePort == 0 {
						svc.Spec.Ports[i].NodePort = port2.NodePort
					}
				}
			}
			// change type to nodeport
			/*if svc.Spec.Type == apiv1.ServiceTypeNodePort &&
				svc.Spec.Type != old.Spec.Type &&
				len(svc.Spec.Ports) > 1 {
				for _, port := range svc.Spec.Ports {
					if port.NodePort == 0 {
						return fmt.Errorf("you cant change service type to nodeport if the service has two or more ports without nodeports!")
					}
				}
			}*/
			//check annotation: // if not existed in new svc, then add it
			updateMap(svc.Annotations, old.Annotations)
		} else {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		kr.gitOpsResList = append(kr.gitOpsResList, svc)
	}
	if commit {
		go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	}
	return nil
}

func (kr *KubeAppRes) deleteService(svcList []*apiv1.Service) error {
	for _, svc := range svcList {
		svc.ObjectMeta.Annotations = labels.AddLabel(svc.ObjectMeta.Annotations, keyword.DELETE_LABLE, keyword.DELETE_LABLE_VALUE)
		//go gitops.CommitK8sResource(kr.cluster, []interface{}{svc})
		kr.gitOpsResList = append(kr.gitOpsResList, svc)
		beego.Warn(fmt.Sprintf("delete service %s successfully!", svc.Name))
	}
	return nil
}

func (kr *KubeAppRes) CreateIngress(ings []*extensions.Ingress, template AppTemplate, commit bool) error {
	for _, ing := range ings {
		//ingIsExist := false
		ingress, err := kr.client.ExtensionsV1beta1().Ingresses(kr.Namespace).Get(ing.Name, metav1.GetOptions{})
		if err == nil {
			//ingIsExist = true
			updateIngressObj(ing, ingress)
		} else {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		if template == nil {
			continue
		}
		kr.gitOpsResList = append(kr.gitOpsResList, ing)
	}
	if commit {
		go gitops.CommitK8sResource(kr.cluster, kr.gitOpsResList)
	}
	return nil
}

func (kr *KubeAppRes) deleteIngress(ings []*extensions.Ingress) error {
	for _, ing := range ings {
		ing.ObjectMeta.Annotations = labels.AddLabel(ing.ObjectMeta.Annotations, keyword.DELETE_LABLE, keyword.DELETE_LABLE_VALUE)
		//go gitops.CommitK8sResource(kr.cluster, []interface{}{ing})
		kr.gitOpsResList = append(kr.gitOpsResList, ing)
		beego.Warn(fmt.Sprintf("delete ingress %s successfully!", ing.Name))
	}
	return nil
}

func updateIngressObj(new, old *extensions.Ingress) {
	new.ResourceVersion = old.ResourceVersion
	// check annotation, if not existed in new ing, then add it
	updateMap(new.Annotations, old.Annotations)
	// check rule, if not existed in new ing, then add it
	new.Spec.Rules = kubeutil.MergeIngressRuleList(new.Spec.Rules, old.Spec.Rules)
	new.Spec.TLS = kubeutil.MergeIngressTLSList(new.Spec.TLS, old.Spec.TLS)
}

func updateMap(new, old map[string]string) {
	for k, v := range old {
		if _, existed := new[k]; !existed {
			new[k] = v
		}
	}
}
