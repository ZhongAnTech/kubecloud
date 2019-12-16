package resource

import (
	"fmt"
	"github.com/astaxie/beego"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/labels"
	"sort"
	// "k8s.io/apimachinery/pkg/fields"

	"kubecloud/backend/service"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/validate"
	"kubecloud/gitops"
)

type ConfigMapVolume struct {
	Name  string   `json:"name,omitempty"`
	Items []string `json:"items,omitempty"`
}

type ConfigMaps []corev1.ConfigMap

func (c ConfigMaps) Len() int { return len(c) }
func (c ConfigMaps) Less(i, j int) bool {
	return c[i].ObjectMeta.CreationTimestamp.Time.After(c[j].ObjectMeta.CreationTimestamp.Time)
}
func (c ConfigMaps) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func ConfigMapCreate(cluster string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if ok, err := configMapValidator(configMap); !ok {
		return nil, err
	}
	go gitops.CommitK8sResource(cluster, []interface{}{configMap})
	check := func(param interface{}) error {
		_, err := ConfigMapInspect(cluster, configMap.Namespace, configMap.Name)
		if errors.IsNotFound(err) {
			return fmt.Errorf("waiting to create......")
		} else {
			return nil
		}
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP*10, SYNC_TIMEOUT*10, check)
	}()
	err := <-result
	return configMap, err
}

func ConfigMapList(cluster string, namespace string) ([]corev1.ConfigMap, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error(fmt.Sprintf("Get ConfigMap list error: %v", err.Error()))
		return nil, common.NewInternalServerError().SetCause(err)
	}
	if namespace == "all" {
		namespace = corev1.NamespaceAll
	}
	configMaps := ConfigMaps{}
	configMapList, err := client.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})
	if err != nil {
		beego.Error(fmt.Sprintf("Get ConfigMap list error: %v", err.Error()))
		return nil, common.NewInternalServerError().SetCause(err)
	}
	configMaps = configMapList.Items
	sort.Sort(configMaps)
	return configMaps, nil
}

func ConfigMapInspect(cluster, namespace, name string) (*corev1.ConfigMap, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error(fmt.Sprintf("Get ConfigMap inspect error: %v", err.Error()))
		return nil, err
	}
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		beego.Error(fmt.Sprintf("Get ConfigMap inspect error: %v", err.Error()))
		return nil, err
	}
	return configMap, nil
}

func ConfigMapUpdate(cluster, namespace, name string, configData *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	configMap, err := ConfigMapInspect(cluster, namespace, name)
	if err != nil {
		beego.Error(fmt.Sprintf("Update ConfigMap error: %v", err.Error()))
		return nil, common.NewInternalServerError().SetCause(err)
	}
	configMap.Data = configData.Data
	if ok, err := configMapValidator(configMap); !ok {
		return nil, err
	}
	go gitops.CommitK8sResource(cluster, []interface{}{configMap})
	return configMap, nil
}

func ConfigMapDelete(cluster, namespace, name string) error {
	configMap, err := ConfigMapInspect(cluster, namespace, name)
	if err != nil {
		beego.Error(fmt.Sprintf("Delete ConfigMap error: %v", err.Error()))
		return common.NewInternalServerError().SetCause(err)
	}
	configMap.ObjectMeta.Annotations = labels.AddLabel(configMap.ObjectMeta.Annotations, keyword.DELETE_LABLE, keyword.DELETE_LABLE_VALUE)
	go gitops.CommitK8sResource(cluster, []interface{}{configMap})
	check := func(param interface{}) error {
		_, err = ConfigMapInspect(cluster, namespace, name)
		if errors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("waiting to delete......")
		}
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP*10, SYNC_TIMEOUT*10, check)
	}()
	err = <-result
	return err
}

func configMapValidator(configMap *corev1.ConfigMap) (bool, error) {
	if validate.IsChineseChar(configMap.Name) {
		return false, common.NewBadRequest().
			SetCode("NotSupportChineseChar").
			SetMessage("not support chinese char")
	}
	for key, _ := range configMap.Data {
		if validate.IsChineseChar(key) {
			return false, common.NewBadRequest().
				SetCode("NotSupportChineseChar").
				SetMessage("not support chinese char")
		}
	}
	return true, nil
}
