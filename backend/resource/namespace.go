package resource

import (
	"fmt"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubecloud/backend/service"
	"kubecloud/common"
	"kubecloud/common/utils"
)

type Namespace struct {
	cluster string
	client  kubernetes.Interface
}

func NewNamespace(cluster string) (*Namespace, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, err
	}
	return &Namespace{
		cluster: cluster,
		client:  client,
	}, nil
}

func (ns *Namespace) CreateNamespace(namespace *corev1.Namespace) (*corev1.Namespace, error) {
	res, err := ns.client.CoreV1().Namespaces().Create(namespace)
	if err != nil {
		return nil, err
	}
	return res, err
}

func (ns *Namespace) NamespaceLabels(namespace string, labels map[string]string) (*corev1.Namespace, error) {
	old, err := ns.client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	old.ObjectMeta.Labels = labels
	new, err := ns.client.CoreV1().Namespaces().Update(old)
	if err != nil {
		return nil, err
	}
	return new, nil
}

func (ns *Namespace) NamespaceDelete(namespace string) (err error) {
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
		lst, err := ns.client.AppsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if n := len(lst.Items); n > 0 {
			ch <- fmt.Sprintf(`deployment count: %v`, n)
		}
	})
	// check pods
	wg.Go(func(...interface{}) {
		lst, err := ns.client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if n := len(lst.Items); n > 0 {
			ch <- fmt.Sprintf(`pod count: %v`, n)
		}
	})
	// check services
	wg.Go(func(...interface{}) {
		lst, err := ns.client.CoreV1().Services(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`service count: %v`, count)
		}
	})
	// check pvcs
	wg.Go(func(...interface{}) {
		lst, err := ns.client.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`persistent volume claim count: %v`, count)
		}
	})
	// check config maps
	wg.Go(func(...interface{}) {
		lst, err := ns.client.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})
		if err != nil {
			setError(err)
		}
		if count := len(lst.Items); count > 0 {
			ch <- fmt.Sprintf(`config map count: %v`, count)
		}
	})
	// check replication controllers
	wg.Go(func(...interface{}) {
		lst, err := ns.client.CoreV1().ReplicationControllers(namespace).List(metav1.ListOptions{})
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
	if err := ns.client.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (ns *Namespace) ListNamespace() (*corev1.NamespaceList, error) {
	res, err := ns.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ns *Namespace) GetNamespace(namespace string) (*corev1.Namespace, error) {
	res, err := ns.client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}
