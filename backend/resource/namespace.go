package resource

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"kubecloud/backend/util/labels"
	"kubecloud/gitops"
	"strings"
	"sync"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"
	"kubecloud/common/validate"

	"github.com/astaxie/beego/orm"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NOTE:
// rename namespace is not allowed by k8s design:
// https://github.com/kubernetes/kubernetes/issues/43867

type NamespaceData struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	CPUQuota    int64  `json:"cpu_quota"`
	MemoryQuota int64  `json:"memory_quota"`
}

type NormalNamespace struct {
	models.K8sNamespace
	CPUQuota    int64 `json:"cpu_quota"`
	MemoryQuota int64 `json:"memory_quota"`
}

func buildResourceQuota(data *NamespaceData) (quota *apiv1.ResourceQuota) {
	quota = &apiv1.ResourceQuota{}
	quota.Name = GenResourceQuotaName(data.Name)
	if data.CPUQuota > 0 {
		quantity := makeKubeResourceQuota(data.CPUQuota, "")
		if quota.Spec.Hard == nil {
			quota.Spec.Hard = make(apiv1.ResourceList)
		}
		quota.Spec.Hard[apiv1.ResourceLimitsCPU] = *quantity
		quota.Spec.Hard[apiv1.ResourceRequestsCPU] = *quantity
	}
	if data.MemoryQuota > 0 {
		quantity := makeKubeResourceQuota(data.MemoryQuota, "Gi")
		if quota.Spec.Hard == nil {
			quota.Spec.Hard = make(apiv1.ResourceList)
		}
		quota.Spec.Hard[apiv1.ResourceLimitsMemory] = *quantity
		quota.Spec.Hard[apiv1.ResourceRequestsMemory] = *quantity
	}
	return quota
}

func NamespaceValidate(data *NamespaceData) error {
	if err := validate.ValidateString(data.Name); err != nil {
		return common.NewBadRequest().
			SetCode("NamespaceInvalidName").
			SetMessage("invalid name").
			SetCause(err)
	}
	return nil
}

func NamespaceCreate(cluster string, data *NamespaceData) (*NormalNamespace, error) {
	// create database item
	if dao.NamespaceExists(cluster, data.Name) {
		err := common.NewConflict().SetCode("NamespaceAlreadyExists").SetMessage("namespace already exists")
		return nil, err
	}
	row := &models.K8sNamespace{
		Cluster:     cluster,
		Name:        data.Name,
		Desc:        data.Desc,
		CPUQuota:    makeKubeResourceQuota(data.CPUQuota, "").String(),
		MemoryQuota: makeKubeResourceQuota(data.MemoryQuota, "Gi").String(),
		AddonsUnix:  models.NewAddonsUnix(),
	}
	if err := dao.NamespaceInsert(row); err != nil {
		err = common.NewInternalServerError().SetCause(err)
		return nil, err
	}
	// create harbor project
	var wg utils.WaitGroup
	var mux sync.Mutex
	res, err := dao.GetAllHarbor(nil)
	if err != nil {
		return nil, err
	}
	harbors := res.List.([]models.ZcloudHarbor)
	for i := range harbors {
		harbor := harbors[i]
		wg.Go(func(...interface{}) {
			client := NewHarborClient(&harbor)
			clientError := client.CreateHarborProject(data.Name)
			if clientError != nil {
				mux.Lock()
				defer mux.Unlock()
				if err == nil {
					err = clientError
				}
			}
		})
	}
	wg.Wait()
	if err != nil {
		return nil, err
	}
	// create namespace
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	err = KubeNamespaceCreate(client, cluster, data.Name)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	// create quota
	quota := buildResourceQuota(data)
	go gitops.CommitK8sResource(cluster, []interface{}{quota})
	ns, err := NamespaceGetOne(cluster, data.Name)
	if err != nil {
		return nil, err
	}
	return switchToNormalNamespace(*ns), nil
}

func NamespaceUpdate(cluster string, data *NamespaceData) (*NormalNamespace, error) {
	row, err := NamespaceGetOne(cluster, data.Name)
	if err != nil {
		return nil, err
	}
	// update desc
	row.Desc = data.Desc
	row.CPUQuota = makeKubeResourceQuota(data.CPUQuota, "").String()
	row.MemoryQuota = makeKubeResourceQuota(data.MemoryQuota, "Gi").String()
	if err := dao.NamespaceUpdate(row); err != nil {
		err = common.NewInternalServerError().SetCause(err)
		return nil, err
	}
	// update quota
	quota := buildResourceQuota(data)
	go gitops.CommitK8sResource(cluster, []interface{}{quota})
	ns, err := NamespaceGetOne(cluster, data.Name)
	if err != nil {
		return nil, err
	}
	return switchToNormalNamespace(*ns), nil
}

func NamespaceDelete(cluster, namespace string) (err error) {
	if _, err := NamespaceGetOne(cluster, namespace); err != nil {
		return err
	}
	client, err := service.GetClientset(cluster)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	var mux sync.Mutex
	setError := func(inputError error) {
		mux.Lock()
		defer mux.Unlock()
		if err != nil {
			err = inputError
		}
	}

	ch := make(chan string)
	var mg utils.WaitGroup
	msgs := []string{}
	mg.Go(func(...interface{}) {
		for {
			msg, ok := <-ch
			if !ok {
				break
			}
			msgs = append(msgs, msg)
		}
	})

	var wg utils.WaitGroup
	// check deployments
	wg.Go(func(...interface{}) {
		lst, err := client.AppsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if n := len(lst.Items); n > 0 {
			ch <- fmt.Sprintf(`deployment count: %v`, n)
		}
	})
	// check pods
	wg.Go(func(...interface{}) {
		lst, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if n := len(lst.Items); n > 0 {
			ch <- fmt.Sprintf(`pod count: %v`, n)
		}
	})
	// check services
	wg.Go(func(...interface{}) {
		lst, err := client.CoreV1().Services(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`service count: %v`, count)
		}
	})
	// check pvcs
	wg.Go(func(...interface{}) {
		lst, err := client.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`persistent volume claim count: %v`, count)
		}
	})
	// check config maps
	wg.Go(func(...interface{}) {
		lst, err := client.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`config map count: %v`, count)
		}
	})
	// check replication controllers
	wg.Go(func(...interface{}) {
		lst, err := client.CoreV1().ReplicationControllers(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`replication controller count count: %v`, count)
		}
	})
	wg.Wait()
	close(ch)
	mg.Wait()
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	if len(msgs) > 0 {
		return common.NewBadRequest().
			SetCode("NamespaceInUse").
			SetMessage("namespace in use").
			SetCause(fmt.Errorf(strings.Join(msgs, ", ")))
	}

	// delete from database
	if err := dao.NamespaceDelete(cluster, namespace); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	// delete harbor projects
	res, err := dao.GetAllHarbor(nil)
	if err != nil {
		return err
	}
	harbors := res.List.([]models.ZcloudHarbor)
	for i := range harbors {
		harbor := harbors[i]
		wg.Go(func(...interface{}) {
			client := NewHarborClient(&harbor)
			clientError := client.DeleteHarborProject(namespace)
			if clientError != nil {
				setError(clientError)
			}
		})
	}
	wg.Wait()
	if err != nil {
		return err
	}
	// delete k8s resource
	k8sNamespace, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	k8sNamespace.ObjectMeta.Annotations = labels.AddLabel(k8sNamespace.ObjectMeta.Annotations, keyword.RESTART_LABLE, keyword.DELETE_LABLE_VALUE)
	go gitops.CommitK8sResource(cluster, []interface{}{k8sNamespace})
	return err
}

func NamespaceGetAll(cluster string, query *utils.FilterQuery) (*utils.QueryResult, error) {
	res, err := dao.NamespaceList([]string{cluster}, nil, query)
	if err != nil {
		return nil, err
	}
	var nsList []NormalNamespace
	list := res.List.([]models.K8sNamespace)
	for _, n := range list {
		item := NormalNamespace{}
		item.K8sNamespace = n
		if len(n.CPUQuota) > 0 {
			quantity, err := k8sresource.ParseQuantity(n.CPUQuota)
			if err != nil {
				beego.Warn(fmt.Sprintf("parse quantity %s failed, %s", n.CPUQuota, err.Error()))
			}
			item.CPUQuota = quantity.Value()
		}
		if len(n.MemoryQuota) > 0 {
			quantity, err := k8sresource.ParseQuantity(n.MemoryQuota)
			if err != nil {
				beego.Warn(fmt.Sprintf("parse quantity %s failed, %s", n.MemoryQuota, err.Error()))
			}
			item.MemoryQuota = quantity.Value() / (1024 * 1024 * 1024)
		}
		nsList = append(nsList, item)
	}
	res.List = nsList
	return res, nil
}

func NamespaceGetOne(cluster, namespace string) (*models.K8sNamespace, error) {
	row, err := dao.NamespaceGet(cluster, namespace)
	if err != nil {
		if err == orm.ErrNoRows {
			err = common.NewNotFound().SetCode("NamespaceNotFound").SetMessage("namespace not found")
		} else {
			err = common.NewInternalServerError().SetCause(err)
		}
	}
	return row, err
}

func KubeNamespaceCreate(client kubernetes.Interface, cluster, name string) error {
	kubeClient := client
	if kubeClient == nil {
		c, err := service.GetClientset(cluster)
		if err != nil {
			return err
		}
		kubeClient = c
	}

	_, err := kubeClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// create
		res := &apiv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"name":                         name,
					keyword.ISTIO_INJECTION_POLICY: keyword.ISTIO_INJECTION_DISABLE,
				},
			},
		}
		go gitops.CommitK8sResource(cluster, []interface{}{res})
		return nil
	}
	logs.Warning("namespace %s already exists in cluster %s", name, cluster)
	return err
}

func makeKubeResourceQuota(number int64, suffix string) *k8sresource.Quantity {
	quantity, _ := k8sresource.ParseQuantity(fmt.Sprintf("%v%s", number, suffix))
	return &quantity
}

func switchToNormalNamespace(ns models.K8sNamespace) *NormalNamespace {
	item := NormalNamespace{}
	item.K8sNamespace = ns
	if len(ns.CPUQuota) > 0 {
		quantity, err := k8sresource.ParseQuantity(ns.CPUQuota)
		if err != nil {
			beego.Warn(fmt.Sprintf("parse quantity %s failed, %s", ns.CPUQuota, err.Error()))
		}
		item.CPUQuota = quantity.Value()
	}
	if len(ns.MemoryQuota) > 0 {
		quantity, err := k8sresource.ParseQuantity(ns.MemoryQuota)
		if err != nil {
			beego.Warn(fmt.Sprintf("parse quantity %s failed, %s", ns.MemoryQuota, err.Error()))
		}
		item.MemoryQuota = quantity.Value() / (1024 * 1024 * 1024)
	}
	return &item
}
