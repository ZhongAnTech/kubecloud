package resource

import (
	"fmt"
	"kubecloud/common/keyword"
	"kubecloud/gitops"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"kubecloud/backend/util/labels"
)

type KubeAppInterface interface {
	CreateOrUpdate(obj interface{}) (interface{}, error)
	Status(appname, podVersion string) (*AppStatus, error)
	Delete(obj interface{}) (interface{}, error)
	AppIsExisted(appname, podVersion string) (bool, error)
	Scale(obj interface{}, replicas int) error
	Restart(obj interface{}) error
	GetOwnerForPod(pod apiv1.Pod, ref *metav1.OwnerReference) interface{}
}

type AppStatus struct {
	ReadyReplicas     int32
	UpdatedReplicas   int32
	AvailableReplicas int32
	AvailableStatus   string
	Message           string
}

type DeploymentRes struct {
	Cluster   string
	Namespace string
	client    kubernetes.Interface
}

func NewDeploymentRes(client kubernetes.Interface, cluster, namespace string) KubeAppInterface {
	return &DeploymentRes{
		Cluster:   cluster,
		Namespace: namespace,
		client:    client,
	}
}

func (kr *DeploymentRes) CreateOrUpdate(obj interface{}) (interface{}, error) {
	dp, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return nil, fmt.Errorf("can not generate deployment object!")
	}
	beego.Info("creating or updating deployment, " + dp.Name)
	//go gitops.CommitK8sResource(kr.Cluster, []interface{}{dp})
	return dp, nil
}

func (kr *DeploymentRes) Status(appname, suffix string) (*AppStatus, error) {
	deployment, err := kr.client.AppsV1beta1().Deployments(kr.Namespace).Get(GenerateDeployName(appname, suffix), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	status := &AppStatus{
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
	}
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == v1beta1.DeploymentAvailable {
			status.AvailableStatus = string(condition.Status)
			status.Message = condition.Message
			break
		}
	}
	return status, nil
}

func (kr *DeploymentRes) Delete(obj interface{}) (interface{}, error) {
	dp, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return nil, fmt.Errorf("can not generate deployment object!")
	}
	dp.ObjectMeta.Annotations = labels.AddLabel(dp.ObjectMeta.Annotations, keyword.DELETE_LABLE, keyword.DELETE_LABLE_VALUE)
	//go gitops.CommitK8sResource(kr.Cluster, []interface{}{dp})
	beego.Warn(fmt.Sprintf("delete deployment %s successfully!", dp.Name))
	return dp, nil
}

func (kr *DeploymentRes) AppIsExisted(appname, suffix string) (bool, error) {
	name := GenerateDeployName(appname, suffix)
	_, err := kr.client.AppsV1beta1().Deployments(kr.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		} else {
			return false, nil
		}
	}
	return true, nil
}

func (kr *DeploymentRes) Scale(obj interface{}, replicas int) error {
	dp, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return fmt.Errorf("can not generate deployment object!")
	}
	num := int32(replicas)
	if *dp.Spec.Replicas == num {
		return nil
	}
	dp.Spec.Replicas = &num
	go gitops.CommitK8sResource(kr.Cluster, []interface{}{dp})
	return nil
}

func (kr *DeploymentRes) Restart(obj interface{}) error {
	dp, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return fmt.Errorf("can not generate deployment object!")
	}
	dp.Spec.Template.ObjectMeta.Annotations = labels.AddLabel(dp.Spec.Template.ObjectMeta.Annotations, keyword.RESTART_LABLE, strconv.FormatInt(time.Now().Unix(), 10))
	go gitops.CommitK8sResource(kr.Cluster, []interface{}{dp})
	return nil
}

func (kr *DeploymentRes) GetOwnerForPod(pod apiv1.Pod, ref *metav1.OwnerReference) interface{} {
	if ref == nil {
		return nil
	}
	rs, err := kr.client.ExtensionsV1beta1().ReplicaSets(pod.Namespace).Get(ref.Name, metav1.GetOptions{})
	if err != nil || rs.UID != ref.UID {
		beego.Warn(fmt.Sprintf("Cannot get replicaset %s for pod %s: %v", ref.Name, pod.Name, err))
		return nil
	}
	// Now find the Deployment that owns that ReplicaSet.
	depRef := metav1.GetControllerOf(rs)
	if depRef == nil {
		return nil
	}
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if depRef.Kind != v1beta1.SchemeGroupVersion.WithKind("Deployment").Kind {
		return nil
	}
	d, err := kr.client.AppsV1beta1().Deployments(pod.Namespace).Get(depRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	if d.UID != depRef.UID {
		return nil
	}
	return d
}
