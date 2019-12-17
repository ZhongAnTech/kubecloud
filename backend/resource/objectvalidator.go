package resource

import (
	"fmt"

	"github.com/astaxie/beego"
	v1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kubecloud/backend/dao"
	"kubecloud/common"
)

type ObjectValidator interface {
	Validator(obj interface{}) error
}

type KubeAppValidator struct {
	namespace string
	client    kubernetes.Interface
}

func NewKubeAppValidator(client kubernetes.Interface, namespace string) *KubeAppValidator {
	return &KubeAppValidator{
		namespace: namespace,
		client:    client,
	}
}

func (validator *KubeAppValidator) Validator(obj interface{}) error {
	var volumes []apiv1.Volume
	if obj == nil {
		return nil
	}
	if dp, ok := obj.(*v1beta1.Deployment); ok {
		volumes = dp.Spec.Template.Spec.Volumes
	}
	for _, vol := range volumes {
		if vol.PersistentVolumeClaim != nil {
			_, err := validator.client.CoreV1().PersistentVolumeClaims(validator.namespace).Get(vol.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return fmt.Errorf("PVC(%s) is not existed in namespace %s!", vol.PersistentVolumeClaim.ClaimName, validator.namespace)
			}
			if err != nil {
				beego.Error("get PVC info failed:", err)
				return fmt.Errorf("get PVC(%s) info failed!", vol.PersistentVolumeClaim.ClaimName)
			}
		}
		if vol.ConfigMap != nil {
			_, err := validator.client.CoreV1().ConfigMaps(validator.namespace).Get(vol.ConfigMap.Name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return fmt.Errorf("Configmap:%s is not existed in namespace %s!", vol.ConfigMap.Name, validator.namespace)
			}
			if err != nil {
				beego.Error("get configmap info failed:", err)
				return fmt.Errorf("get configmap:%s info failed!", vol.ConfigMap.Name)
			}
		}
	}
	return nil
}

type KubeSvcValidator struct {
	cluster   string
	namespace string
	appname   string
}

func NewKubeSvcValidator(cluster, namespace, appname string) *KubeSvcValidator {
	return &KubeSvcValidator{
		cluster:   cluster,
		namespace: namespace,
		appname:   appname,
	}
}

func (validator *KubeSvcValidator) Validator(obj interface{}) error {
	if obj == nil {
		return nil
	}
	svc, ok := obj.(*apiv1.Service)
	if !ok {
		return nil
	}
	for _, port := range svc.Spec.Ports {
		if err := checkNodePort(validator.cluster, validator.appname, int(port.NodePort)); err != nil {
			return err
		}
	}

	return nil
}

func checkNodePort(cluster, newAppname string, nodePort int) error {
	if nodePort != 0 && cluster != "" {
		svcPorts, err := dao.NewK8sServiceModel().List(cluster, common.AllNamespace, nodePort)
		if err != nil {
			return err
		}
		for _, item := range svcPorts {
			if item.Service.OwnerName != newAppname {
				return fmt.Errorf("NodePort(%d) is used by application(%s/%s/%s)!",
					nodePort, cluster, item.Namespace, item.Service.OwnerName)
			}
		}
	}
	return nil
}
