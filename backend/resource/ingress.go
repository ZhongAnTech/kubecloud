package resource

import (
	"fmt"
	"strconv"
	"strings"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/backend/util/kubeutil"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"
	"kubecloud/common/validate"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	PROTOCOL_HTTPS = "https"
	PROTOCOL_HTTP  = "http"
)

type BackendServer struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

type IngressDetail struct {
	Backend []BackendServer `json:"backend"`
	Config  IngressConfig   `json:"config,omitempty"`
}

type IngressPath struct {
	Path string `json:"path,omitempty"`
	// ServiceName is equal application name, not real service name in k8s
	ServiceName string `json:"service_name,omitempty"`
	ServicePort int    `json:"service_port,omitempty"`
}

type Ingress struct {
	Name          string        `json:"name"`
	Namespace     string        `json:"namespace"`
	Protocol      string        `json:"protocol,omitempty"`
	SecretName    string        `json:"secret_name,omitempty"`
	Host          string        `json:"host,omitempty"`
	Paths         []IngressPath `json:"paths"`
	IngressConfig `json:",inline"`
}

type IngressRule struct {
	models.K8sIngressRule `json:",inline"`
	Protocol              string `json:"protocol"`
	CreateAt              string `json:"create_at"`
	UpdateAt              string `json:"update_at"`
	DeleteAt              string `json:"delete_at"`
}

type SimpleIngressRule struct {
	Host  string                    `json:"host,omitempty"`
	Paths []v1beta1.HTTPIngressPath `json:"paths,omitempty"`
}

type SimpleIngressDetail struct {
	Name  string               `json:"name"`
	TLS   []v1beta1.IngressTLS `json:"tls,omitempty"`
	Rules []SimpleIngressRule  `json:"rules,omitempty"`
}

type IngressRes struct {
	cluster       string
	client        kubernetes.Interface
	ModelHandle   *dao.K8sIngressModel
	modelEndpoint *dao.K8sEndpointModel
	modelSvc      *dao.K8sServiceModel
	kubeRule      *dao.IngressRuleModel
	listNSFunc    NamespaceListFunction
}

func NewEmptyIngress(namespace string) Ingress {
	ingress := Ingress{}
	ingress.Namespace = namespace

	return ingress
}

func NewIngressRes(cluster string, client kubernetes.Interface, get NamespaceListFunction) (*IngressRes, error) {
	ingHandle := &IngressRes{
		cluster:    cluster,
		listNSFunc: get,
	}
	var err error
	newclient := client
	if newclient == nil {
		newclient, err = service.GetClientset(cluster)
		if err != nil {
			return nil, fmt.Errorf("get client error %v", err)
		}
	}
	ingHandle.client = newclient
	ingHandle.ModelHandle = dao.NewK8sIngressModel()
	ingHandle.modelEndpoint = dao.NewK8sEndpointModel()
	ingHandle.modelSvc = dao.NewK8sServiceModel()
	ingHandle.kubeRule = dao.NewIngressRuleModel()

	return ingHandle, nil
}

func (ing *IngressRes) Validate(ingress *Ingress) error {
	err := validate.ValidateString(ingress.Namespace)
	if err != nil {
		return err
	}
	if ingress.Protocol == PROTOCOL_HTTPS {
		if ingress.SecretName == "" {
			return fmt.Errorf("https must have certificate secret name!")
		}
		// todo check secret is existed or not
	} else if ingress.Protocol != PROTOCOL_HTTP {
		return fmt.Errorf("communication protocol must be http or https!")
	}
	err = validate.ValidateDomainName(ingress.Host)
	if err != nil {
		return err
	}
	if len(ingress.Paths) == 0 {
		return fmt.Errorf("path and its backend server must be given!")
	}
	for i, path := range ingress.Paths {
		if len(path.Path) > validate.NormalMaxLen {
			return fmt.Errorf("path length can not be above %v bytes!", validate.NormalMaxLen)
		}
		//check path
		for j, item := range ingress.Paths {
			if i != j && utils.PathsIsEqual(path.Path, item.Path) {
				return fmt.Errorf("the paths you specified are duplicated!")
			}
		}
		if path.Path != "" {
			ingress.Paths[i].Path = utils.AddRootPath(ingress.Paths[i].Path)
		}
		//check service
		errs := validation.IsQualifiedName(path.ServiceName)
		if len(errs) != 0 {
			return fmt.Errorf(strings.Join(errs, ";"))
		}
		errs = validation.IsValidPortNum(path.ServicePort)
		if len(errs) != 0 {
			return fmt.Errorf("server port %v is not right!", path.ServicePort)
		}
		// check service and port is existed
		if err != nil {
			return err
		}
		svc, err := dao.NewK8sServiceModel().Get(ing.cluster, ingress.Namespace, "", path.ServiceName)
		if err != nil {
			if err == orm.ErrNoRows {
				return fmt.Errorf("server(%s/%s/%s) is not existed!", ing.cluster, ingress.Namespace, path.ServiceName)
			}
			return err
		}
		// set real svc name
		ingress.Paths[i].ServiceName = svc.Name
		isExisted := false
		for _, item := range svc.Ports {
			if item.Port == path.ServicePort {
				isExisted = true
				break
			}
		}
		if !isExisted {
			return fmt.Errorf("server(%s/%s/%s) port %v is not existed!", ing.cluster, ingress.Namespace, path.ServiceName, path.ServicePort)
		}
	}

	return nil
}

func (ing *IngressRes) makeKubeIngress(ingress Ingress) *v1beta1.Ingress {
	kubeIngress := &v1beta1.Ingress{}
	kubeIngress.Kind = "Ingress"
	kubeIngress.Name = ingress.Name
	kubeIngress.Namespace = ingress.Namespace
	kubeIngress.Labels = map[string]string{}
	kubeIngress.Annotations = map[string]string{}
	if ingress.Protocol == PROTOCOL_HTTPS {
		tls := v1beta1.IngressTLS{
			Hosts:      []string{ingress.Host},
			SecretName: ingress.SecretName,
		}
		kubeIngress.Spec.TLS = append(kubeIngress.Spec.TLS, tls)
	}

	http := v1beta1.HTTPIngressRuleValue{}
	for _, item := range ingress.Paths {
		path := v1beta1.HTTPIngressPath{
			Path: item.Path,
			Backend: v1beta1.IngressBackend{
				ServiceName: item.ServiceName,
				ServicePort: intstr.FromInt(item.ServicePort),
			},
		}
		http.Paths = append(http.Paths, path)
	}
	iRule := v1beta1.IngressRule{
		Host: ingress.Host,
	}
	iRule.HTTP = &http
	kubeIngress.Spec.Rules = append(kubeIngress.Spec.Rules, iRule)

	return kubeIngress
}

func (ing *IngressRes) getBackend(rule *models.K8sIngressRule) ([]BackendServer, error) {
	empty := []BackendServer{}
	svc, err := ing.modelSvc.Get(ing.cluster, rule.Namespace, "", rule.ServiceName)
	if err != nil {
		return empty, err
	}
	targetPort := -1
	for _, port := range svc.Ports {
		if port.Port == rule.ServicePort {
			targetPort = port.TargetPort
		}
	}
	if targetPort == -1 {
		return empty, fmt.Errorf("service port %v is not existed in service ports", rule.ServicePort)
	}
	ep, err := ing.modelEndpoint.Get(ing.cluster, rule.Namespace, rule.ServiceName, int32(targetPort))
	if err != nil {
		beego.Debug("get endpoint ", rule.ServiceName, "failed: ", err)
		return empty, err
	}
	return getBackendServer(svc, ep), nil
}

func (ing *IngressRes) CreateIngress(ingress *Ingress) error {
	if ingress.Name == "" {
		ingress.Name = ing.kubeRule.GetIngressNameByHost(ing.cluster, ingress.Namespace, ingress.Host)
		if ingress.Name == "" {
			ingress.Name = genIngressName(ing.cluster, *ingress)
		}
	}
	if err := ing.kubeRule.CheckHostUnique(ing.cluster, ingress.Namespace, ingress.Host); err != nil {
		return common.NewConflict().SetCause(err)
	}
	checkPaths := []string{}
	for _, path := range ingress.Paths {
		checkPaths = append(checkPaths, path.Path)
	}
	if err := ing.kubeRule.CheckPathsUniqueInHost(ing.cluster, ingress.Namespace, ingress.Host, checkPaths, -1); err != nil {
		return common.NewConflict().SetCause(err)
	}
	var obj *v1beta1.Ingress
	create := true
	old, err := ing.client.ExtensionsV1beta1().Ingresses(ingress.Namespace).Get(ingress.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return common.NewInternalServerError().SetCause(err)
		}
		// create a new ingress
		obj = ing.makeKubeIngress(*ingress)
	} else {
		// update
		obj, err = ing.addRule(*ingress, old)
		if err != nil {
			return common.NewConflict().SetCause(err)
		}
		create = false
	}
	var paths []string
	svcList := make(map[string]interface{})
	for _, path := range ingress.Paths {
		paths = append(paths, path.Path)
		svcList[path.ServiceName] = nil
	}
	confer := NewIngressConfer(&ingress.IngressConfig, paths, ingress.Host)
	if err = confer.Validate(); err != nil {
		return common.NewBadRequest().SetCause(err)
	}
	if err = confer.SetIngress(obj.Annotations); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	if create {
		_, err = ing.client.ExtensionsV1beta1().Ingresses(obj.Namespace).Create(obj)
	} else {
		_, err = ing.client.ExtensionsV1beta1().Ingresses(obj.Namespace).Update(kubeutil.DeleteCreatedDefaultAnno(obj))
	}
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	// set svc
	for svc, _ := range svcList {
		if newerr := ing.setService(ingress.Namespace, svc, confer); newerr != nil {
			beego.Warn("set service annotations failed: ", newerr)
		}
	}
	check := func(param interface{}) error {
		for _, path := range ingress.Paths {
			record, err := ing.ModelHandle.Get(ing.cluster, ingress.Namespace, ingress.Name)
			if err != nil {
				return err
			}
			found := false
			for _, rule := range record.Rules {
				if ingress.Host == rule.Host && path.Path == rule.Path {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("ingess rule(%s/%s) is not existed!", ingress.Host, path.Path)
			}
		}
		return nil
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP, SYNC_TIMEOUT, check)
	}()
	err = <-result
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	return nil
}

func (ing *IngressRes) UpdateIngress(id int64, ingress Ingress) error {
	oldRule, err := ing.kubeRule.GetByID(ing.cluster, ingress.Namespace, id)
	if err != nil {
		if err == orm.ErrNoRows {
			return common.NewNotFound().SetCause(fmt.Errorf(`ingress "%v" not found`, id))
		}
		return common.NewInternalServerError().SetCause(err)
	}
	if ingress.Host != oldRule.Host {
		return common.NewBadRequest().SetCause(fmt.Errorf("you can not modify your host"))
	}
	// check path
	checkPaths := []string{}
	for _, path := range ingress.Paths {
		checkPaths = append(checkPaths, path.Path)
	}
	if err := ing.kubeRule.CheckPathsUniqueInHost(ing.cluster, ingress.Namespace, ingress.Host, checkPaths, id); err != nil {
		return common.NewConflict().SetCause(err)
	}
	if ingress.Name == "" {
		ingress.Name = oldRule.IngressName
	}
	//ingessObj := ing.makeKubeIngress(ingress)
	old, err := ing.client.ExtensionsV1beta1().Ingresses(ingress.Namespace).Get(ingress.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return common.NewNotFound().
				SetCause(fmt.Errorf(`ingress "/%s/%s/%s" is not found`, ing.cluster, ingress.Namespace, ingress.Name))
		}
		return common.NewInternalServerError().SetCause(err)
	}
	obj, err := ing.modifyRule(ingress, old, *oldRule)
	if err != nil {
		return common.NewBadRequest().SetCause(err)
	}
	if obj.Annotations == nil {
		obj.Annotations = make(map[string]string)
	}
	// update
	var paths []string
	svcList := make(map[string]interface{})
	for _, path := range ingress.Paths {
		paths = append(paths, path.Path)
		svcList[path.ServiceName] = nil
	}
	confer := NewIngressConfer(&ingress.IngressConfig, paths, ingress.Host)
	if err = confer.Validate(); err != nil {
		return common.NewBadRequest().SetCause(err)
	}
	if err = confer.SetIngress(obj.Annotations); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	if !utils.ObjectIsEqual(old, obj) {
		if _, err = ing.client.ExtensionsV1beta1().Ingresses(obj.Namespace).Update(kubeutil.DeleteCreatedDefaultAnno(obj)); err != nil {
			return common.NewInternalServerError().SetCause(err)
		}
	}
	// set svc
	for svc, _ := range svcList {
		if newerr := ing.setService(ingress.Namespace, svc, confer); newerr != nil {
			beego.Warn("set service annotations failed: ", newerr)
		}
	}
	return nil
}

func (ing *IngressRes) DeleteIngress(namespace string, id int64) error {
	rule, err := ing.kubeRule.GetByID(ing.cluster, namespace, id)
	if err != nil {
		return err
	}
	old, err := ing.client.ExtensionsV1beta1().Ingresses(rule.Namespace).Get(rule.Ingress.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			beego.Warn(fmt.Sprintf("ingress(%s/%s/%s) is not found, so delete its info from db", ing.cluster, namespace, rule.Ingress.Name))
			err = ing.ModelHandle.Delete(rule.Cluster, rule.Namespace, rule.Ingress.Name)
			if err == orm.ErrNoRows {
				return nil
			}
		}
		return err
	}
	obj, err := ing.deleteRule(*rule, old)
	if err != nil {
		return err
	}
	if len(obj.Spec.Rules) == 0 {
		// delete
		err = ing.client.ExtensionsV1beta1().Ingresses(rule.Namespace).Delete(rule.Ingress.Name, &metav1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
		}
	} else {
		// update
		_, err = ing.client.ExtensionsV1beta1().Ingresses(obj.Namespace).Update(kubeutil.DeleteCreatedDefaultAnno(obj))
	}
	if err != nil {
		return err
	}
	check := func(param interface{}) error {
		_, err := ing.kubeRule.GetByID(ing.cluster, namespace, id)
		if err == orm.ErrNoRows {
			return nil
		}
		return err
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP, SYNC_TIMEOUT, check)
	}()
	err = <-result
	return err
}

func (ing *IngressRes) ListIngresses(cluster, namespace, svcname string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	nslist := []string{}
	if namespace != common.AllNamespace {
		nslist = append(nslist, namespace)
	} else {
		nslist = ing.listNSFunc()
	}
	if len(nslist) == 0 {
		return utils.InitQueryResult([]models.K8sIngressRule{}, filterQuery), nil
	}
	res, err := ing.kubeRule.List(cluster, nslist, svcname, filterQuery)
	if err != nil {
		beego.Error(err)
		return nil, err
	}
	items, ok := res.List.([]models.K8sIngressRule)
	if !ok {
		return nil, fmt.Errorf("data type is not right!")
	}
	list := []IngressRule{}
	svcHandler := dao.NewK8sServiceModel()
	for _, item := range items {
		svcName := svcname
		svcPort := item.ServicePort
		if svcName == "" {
			svc, err := svcHandler.Get(cluster, namespace, "", item.ServiceName)
			if err != nil {
				svcPort = 0
				beego.Warn("get svc for ingress failed:", err, item)
			} else {
				svcName = svc.Name
			}
		}
		one := IngressRule{}
		one.K8sIngressRule = item
		one.Path = utils.GetRootPath(one.Path)
		one.ServiceName = svcName
		one.ServicePort = svcPort
		if item.IsTls {
			one.Protocol = PROTOCOL_HTTPS
		} else {
			one.Protocol = PROTOCOL_HTTP
		}
		one.CreateAt = item.CreateAt.Format("2006-01-02 15:04:05")
		one.UpdateAt = item.UpdateAt.Format("2006-01-02 15:04:05")
		one.DeleteAt = item.DeleteAt.Format("2006-01-02 15:04:05")
		list = append(list, one)
	}
	res.List = list

	return res, nil
}

func (ing *IngressRes) GetIngressDetail(namespace string, id int64) (*IngressDetail, error) {
	rule, err := ing.kubeRule.GetByID(ing.cluster, namespace, id)
	if err != nil {
		beego.Error(err)
		return nil, err
	}
	be, err := ing.getBackend(rule)
	if err != nil {
		beego.Error(err)
		be = []BackendServer{}
	}
	detail := &IngressDetail{
		Backend: be,
		Config:  ing.GetConfig(rule),
	}

	return detail, nil
}

func (ing *IngressRes) GetSimpleIngressDetail(namespace, appname string) ([]SimpleIngressDetail, error) {
	listopt, _ := GetListOption(keyword.LABEL_APPNAME_KEY, appname)
	// Ingress
	inglist, err := ing.client.ExtensionsV1beta1().Ingresses(namespace).List(listopt)
	if err != nil {
		beego.Error("Get ingress information failed: " + err.Error())
		if !errors.IsNotFound(err) {
			return nil, err
		}
	}
	var ilist []SimpleIngressDetail
	getSimpleIngressDetail := func(kubeing v1beta1.Ingress) SimpleIngressDetail {
		var detail SimpleIngressDetail

		var rules []SimpleIngressRule
		for _, path := range kubeing.Spec.Rules {
			rule := SimpleIngressRule{}
			rule.Host = path.Host
			rule.Paths = path.HTTP.Paths
			rules = append(rules, rule)
		}

		detail.Rules = rules
		detail.Name = kubeing.Name
		detail.TLS = kubeing.Spec.TLS

		return detail
	}
	for _, item := range inglist.Items {
		ilist = append(ilist, getSimpleIngressDetail(item))
	}

	return ilist, nil
}

func (ing *IngressRes) GetIngressRuleByID(namespace string, id int64) (*models.K8sIngressRule, error) {
	return ing.kubeRule.GetByID(ing.cluster, namespace, id)
}

func (ing *IngressRes) GetConfig(rule *models.K8sIngressRule) IngressConfig {
	config := IngressConfig{}
	if rule == nil {
		return config
	}
	svcAnno := make(map[string]string)
	svc, err := ing.modelSvc.Get(rule.Cluster, rule.Namespace, "", rule.ServiceName)
	if err == nil && svc.Annotation != "" {
		utils.SimpleJsonUnmarshal(svc.Annotation, &svcAnno)
	}
	ingAnno := make(map[string]string)
	if rule.Ingress.Annotation != "" {
		utils.SimpleJsonUnmarshal(rule.Ingress.Annotation, &ingAnno)
	}
	confer := NewIngressConfer(nil, []string{rule.Path}, rule.Host)
	config = confer.GetConfig(ingAnno, svcAnno)
	//set tls
	config.Protocol = PROTOCOL_HTTP
	if rule.IsTls {
		config.Protocol = PROTOCOL_HTTPS
		config.SecretName = rule.SecretName
	}

	return config
}

func (ing *IngressRes) addRule(base Ingress, old *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	obj := old.DeepCopy()
	ir := -1
	for i, r := range old.Spec.Rules {
		if r.Host == base.Host {
			ir = i
			if r.HTTP == nil {
				break
			}
			for _, path := range r.HTTP.Paths {
				for _, item := range base.Paths {
					if utils.PathsIsEqual(path.Path, item.Path) {
						return nil, fmt.Errorf("host(%s) and path(%s) is existed in current ingress", base.Host, item.Path)
					}
				}
			}
			break
		}
	}
	single := ing.makeKubeIngress(base)
	// add rule
	if ir == -1 {
		obj.Spec.Rules = append(obj.Spec.Rules, single.Spec.Rules...)
	} else {
		// just append a path and server
		if obj.Spec.Rules[ir].HTTP == nil {
			obj.Spec.Rules[ir].HTTP = single.Spec.Rules[0].HTTP
		} else {
			obj.Spec.Rules[ir].HTTP.Paths = append(obj.Spec.Rules[ir].HTTP.Paths, single.Spec.Rules[0].HTTP.Paths...)
		}
	}
	// modify or add a tls
	if base.Protocol == PROTOCOL_HTTPS {
		obj.Spec.TLS = modifyTLS(obj.Spec.TLS, base.Host, base.SecretName)
	} else {
		obj.Spec.TLS = kubeutil.DeleteHostFromTLS(obj.Spec.TLS, base.Host)
	}
	return obj, nil
}

// modify
func (ing *IngressRes) modifyRule(base Ingress, old *v1beta1.Ingress, oldrule models.K8sIngressRule) (*v1beta1.Ingress, error) {
	obj := old.DeepCopy()
	ir := -1
	for i, r := range obj.Spec.Rules {
		if r.Host == base.Host {
			ir = i
			break
		}
	}
	if ir < 0 {
		return nil, fmt.Errorf("host(%s) is not existed in current ingress", base.Host)
	}
	// modify rule
	http := obj.Spec.Rules[ir].HTTP
	for _, item := range base.Paths {
		found := false
		for i, path := range http.Paths {
			if utils.PathsIsEqual(path.Path, item.Path) {
				// update backend
				if path.Backend.ServiceName != item.ServiceName ||
					path.Backend.ServicePort.IntValue() != item.ServicePort {
					http.Paths[i].Backend.ServiceName = item.ServiceName
					http.Paths[i].Backend.ServicePort = intstr.FromInt(item.ServicePort)
				}
				found = true
				break
			}
		}
		if !found {
			// append a new path
			http.Paths = append(http.Paths, v1beta1.HTTPIngressPath{
				Path: item.Path,
				Backend: v1beta1.IngressBackend{
					ServiceName: item.ServiceName,
					ServicePort: intstr.FromInt(item.ServicePort),
				},
			})
		}
	}

	// delete old path if need
	deletePath := oldrule.Path
	for _, item := range base.Paths {
		if utils.PathsIsEqual(item.Path, oldrule.Path) {
			deletePath = ""
			break
		}
	}
	if deletePath != "" {
		paths := []v1beta1.HTTPIngressPath{}
		for _, path := range http.Paths {
			if path.Path != deletePath {
				paths = append(paths, path)
			}
		}
		http.Paths = paths
		// delete path's rate limit config
		DeletePathRateLimit(obj.Annotations, []string{deletePath})
	}

	// modify tls
	if base.Protocol == PROTOCOL_HTTPS {
		obj.Spec.TLS = modifyTLS(obj.Spec.TLS, base.Host, base.SecretName)
	} else {
		obj.Spec.TLS = kubeutil.DeleteHostFromTLS(obj.Spec.TLS, base.Host)
	}
	return obj, nil
}

// deleteRule from memory
func (ing *IngressRes) deleteRule(rule models.K8sIngressRule, old *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	obj := old.DeepCopy()
	obj.Spec.Rules = []v1beta1.IngressRule{}
	index := -1
	// delete rule
	for i, r := range old.Spec.Rules {
		save := true
		if r.Host == rule.Host && index < 0 && r.HTTP != nil {
			for _, path := range r.HTTP.Paths {
				if utils.PathsIsEqual(path.Path, rule.Path) {
					index = i
					save = false
					break
				}
			}
		}
		if save {
			obj.Spec.Rules = append(obj.Spec.Rules, r)
		}
	}
	// update rule
	if index < 0 {
		// not found
		beego.Warn("rule:", rule, "is not found in ingress resource, so delete it from db!")
		return old, ing.ModelHandle.DeleteRule(rule)
	}
	destRule := old.Spec.Rules[index]
	if destRule.HTTP != nil {
		// delete path
		paths := []v1beta1.HTTPIngressPath{}
		for _, path := range destRule.HTTP.Paths {
			if !utils.PathsIsEqual(path.Path, rule.Path) {
				paths = append(paths, path)
			}
		}
		if len(destRule.HTTP.Paths) == len(paths) && len(paths) != 0 {
			return nil, fmt.Errorf("path(%s) of host(%s) is not existed!", utils.GetRootPath(rule.Path), rule.Host)
		}
		if len(paths) != 0 {
			// just delete rule.Path from destRule
			destRule.HTTP.Paths = paths
			obj.Spec.Rules = append(obj.Spec.Rules, destRule)
		}
		// delete path's rate limit config
		DeletePathRateLimit(obj.Annotations, []string{rule.Path})
	}

	if rule.SecretName == "" || len(destRule.HTTP.Paths) != 0 {
		return obj, nil
	}
	// delete host from tls
	// if tls.hosts is empty, delete tls from Spec.TLS
	obj.Spec.TLS = kubeutil.DeleteHostFromTLS(old.Spec.TLS, rule.Host)
	return obj, nil
}

func (ing *IngressRes) setService(namespace, svcname string, confer *IngressConfer) error {
	svc, err := ing.client.CoreV1().Services(namespace).Get(svcname, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if confer == nil {
		return nil
	}
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	if err = confer.SetService(svc.Annotations); err != nil {
		return err
	}
	if confer.config.Affinity != nil {
		if confer.config.Affinity.Affinity {
			svc.Spec.SessionAffinity = apiv1.ServiceAffinityClientIP
		} else {
			svc.Spec.SessionAffinity = apiv1.ServiceAffinityNone
		}
	}
	_, err = ing.client.CoreV1().Services(svc.Namespace).Update(svc)
	return err
}

func modifyTLS(TLS []v1beta1.IngressTLS, host, secret string) []v1beta1.IngressTLS {
	tlsList := []v1beta1.IngressTLS{}
	index := -1
	for it, tls := range TLS {
		if tls.SecretName == secret {
			exist := false
			for _, item := range tls.Hosts {
				if item == host {
					exist = true
					break
				}
			}
			if !exist {
				// just add host to tls.hosts
				TLS[it].Hosts = append(TLS[it].Hosts, host)
			}
			index = it
			break
		}
	}
	if index < 0 && secret != "" {
		// just add a new tls
		TLS = append(TLS, v1beta1.IngressTLS{
			Hosts:      []string{host},
			SecretName: secret,
		})
	}
	// delete host from tls which secret name is not secret
	// if tls.hosts is empty , delete tls from obj.Spec.TLS
	for _, tls := range TLS {
		if secret != tls.SecretName {
			hosts := []string{}
			for _, item := range tls.Hosts {
				if item != host {
					hosts = append(hosts, item)
				}
			}
			if len(hosts) != len(tls.Hosts) {
				if len(hosts) != 0 {
					tls.Hosts = hosts
					tlsList = append(tlsList, tls)
				}
			} else {
				tlsList = append(tlsList, tls)
			}
		} else {
			tlsList = append(tlsList, tls)
		}
	}
	return tlsList
}

func genIngressName(cluster string, ingress Ingress) string {
	c, _ := dao.GetCluster(cluster)
	if c != nil {
		if strings.HasSuffix(ingress.Host, c.DomainSuffix) {
			return fmt.Sprintf("ing-%s", getIngressNameSuffix(ingress.Host, c.DomainSuffix))
		}
	}
	return fmt.Sprintf("external-%s", getIngressNameSuffix(ingress.Host, ""))
}

func getIngressNameSuffix(host, suffix string) string {
	items := strings.Split(strings.TrimSuffix(host, "."+suffix), ".")
	if len(items) > 1 && suffix == "" {
		return strings.Join(items[0:len(items)-1], "-")
	}
	return strings.Join(items, "-")
}

// calc pod weight
func getVersionFromPodName(name string) string {
	splits := strings.Split(name, "-")
	if len(splits) <= 3 {
		return ""
	}
	return splits[len(splits)-3]
}

func getSvcVersionWeight(svcAnns map[string]string) (bool, map[string]int) {
	weightMap := make(map[string]int)
	hasWeight := false
	for k, _ := range svcAnns {
		if strings.HasPrefix(k, IngressWeightAnnotationKeyPre) {
			weight := utils.GetLabelIntValue(svcAnns, k, 0)
			weightMap[strings.TrimPrefix(k, IngressWeightAnnotationKeyPre)] = weight
			hasWeight = true
		}
	}
	return hasWeight, weightMap
}

func getPodNumMap(addrList []*models.K8sEndpointAddress, weightMap map[string]int) (map[string]int, int) {
	// pvn: pod:version-num
	pvnMap := make(map[string]int)
	otherPodNum := 0
	if weightMap == nil {
		return pvnMap, len(addrList)
	}
	// calc pods num for given pod version
	for _, address := range addrList {
		if address.TargetRefName != "" {
			version := getVersionFromPodName(address.TargetRefName)
			if _, existed := weightMap[version]; existed {
				pvnMap[version]++
			} else {
				otherPodNum++
			}
		} else {
			otherPodNum++
		}
	}
	return pvnMap, otherPodNum
}

func getWeightMap(verWeight map[string]int, pvnMap map[string]int) (map[string]int, int) {
	weightMap := make(map[string]int)
	otherWeight := 0
	weightCount := 0
	for ver, v := range verWeight {
		for pver, _ := range pvnMap {
			if ver == pver {
				weightMap[ver] = v
				weightCount += v
			}
		}
	}
	if weightCount <= models.MAX_WEIGHT {
		otherWeight = models.MAX_WEIGHT - weightCount
	}
	return weightMap, otherWeight
}

func getBackendServer(svc *models.K8sService, endpoint *models.K8sEndpoint) []BackendServer {
	svcAnnos := make(map[string]string)
	utils.SimpleJsonUnmarshal(svc.Annotation, &svcAnnos)
	hasWeight, verWeight := getSvcVersionWeight(svcAnnos)
	// calc pods num for given pod version
	podsMap, otherPodNum := getPodNumMap(endpoint.Addresses, verWeight)
	weightMap, otherWeight := getWeightMap(verWeight, podsMap)
	protocol := PROTOCOL_HTTP
	if endpoint.Port == 443 {
		protocol = PROTOCOL_HTTPS
	}
	bsList := []BackendServer{}
	for _, address := range endpoint.Addresses {
		url := protocol + "://" + address.IP + ":" + strconv.Itoa(int(endpoint.Port))
		name := url
		if address.TargetRefName != "" {
			name = address.TargetRefName
		}
		version := getVersionFromPodName(name)
		weight := models.MAX_WEIGHT / len(endpoint.Addresses)
		if hasWeight {
			svcWeight := otherWeight
			if w, ok := weightMap[version]; ok {
				svcWeight = w
			}
			podsNum := otherPodNum
			if num, ok := podsMap[version]; ok {
				podsNum = num
			}
			if svcWeight == models.MIN_WEIGHT {
				weight = models.MIN_WEIGHT
			} else {
				if models.MIN_WEIGHT < svcWeight && svcWeight <= models.MAX_WEIGHT && podsNum != 0 {
					pw := svcWeight / podsNum
					if pw != 0 {
						weight = pw
					}
				}
			}
		}
		bsList = append(bsList, BackendServer{
			Name:   name,
			URL:    url,
			Weight: weight,
		})
	}
	return bsList
}
