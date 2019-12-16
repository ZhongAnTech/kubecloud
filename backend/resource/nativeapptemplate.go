package resource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"kubecloud/backend/models"
	"kubecloud/backend/util/kubeutil"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"
	"kubecloud/common/validate"

	"github.com/astaxie/beego"
	v1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "k8s.io/apiserver/pkg/storage/names"
)

// native app template and api
type NativeAppTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Deployment        *v1beta1.Deployment   `json:"deployment,omitempty"`
	Services          []*apiv1.Service      `json:"services,omitempty"`
	Ingresses         []*extensions.Ingress `json:"ingresses,omitempty"`
	Config            DeployConfig          `json:"config"`
}

//context is file context of native template,
func CreateNativeAppTemplate(app models.ZcloudApplication, context string, config interface{}) (AppTemplate, error) {
	native := &NativeAppTemplate{}
	if err := json.Unmarshal([]byte(app.Template), native); err != nil {
		return nil, err
	}
	if context != "" {
		template := NewNativeTemplate()
		template.Config = native.Config
		if err := json.Unmarshal([]byte(context), template); err != nil {
			return nil, err
		}
		if conf, ok := config.(*DeployConfig); ok && conf != nil {
			template.Config = *conf
		}
		if err := template.Validate(); err != nil {
			return nil, err
		}
		tplList, _, err := template.GenNativeAppTemplate(app.Namespace, app.Name)
		if err != nil {
			return nil, err
		}
		if len(tplList) > 0 {
			native = tplList[0]
		} else {
			return nil, fmt.Errorf("template context has no application object!")
		}
	}
	return native, nil
}

func (tp *NativeAppTemplate) GenerateAppObject(cluster, namespace, tplname, domainSuffix string) (*models.ZcloudApplication, error) {
	app := &models.ZcloudApplication{
		Id:                0,
		Cluster:           cluster,
		Namespace:         namespace,
		Name:              tp.GetAppName(),
		Kind:              tp.GetAppKind(),
		TemplateName:      tplname,
		LabelSelector:     tp.getAppLabelSelector(),
		BaseName:          "",
		DomainName:        "",
		InjectServiceMesh: strconv.FormatBool(tp.Config.InjectServiceMesh),
		PodVersion:        v1.SimpleNameGenerator.GenerateName(""),
	}
	bItem, err := tp.replaceImagePullAddr(cluster).String()
	if err != nil {
		return nil, err
	}
	mainImage := ""
	var containers []apiv1.Container
	replicas := int32(default_replicas)
	if tp.GetAppKind() == AppKindDeployment {
		containers = tp.Deployment.Spec.Template.Spec.Containers
		if tp.Deployment.Spec.Replicas != nil {
			replicas = *tp.Deployment.Spec.Replicas
		}
	}
	if len(containers) > 0 {
		mainImage = containers[0].Image
	}
	if tp.Config.Version == "" {
		tp.Config.Version = GetResourceVersion(mainImage, ResTypeImage, "")
	}
	if err := validate.ValidateAppVersion(tp.Config.Version); err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.Name == app.Name {
			// main container
			mainImage = container.Image
			break
		}
	}
	app.Image = mainImage
	app.Replicas = int(replicas)
	app.Template = bItem
	app.Version = tp.Config.Version
	domainName := app.DomainName
	if domainName == "" {
		domainName = GenerateDomainName(app.Name)
	}
	app.DefaultDomainAddr = GenerateIngressHost(domainName, domainSuffix, "")
	return app, nil
}

func (tp *NativeAppTemplate) UpdateAppObject(app *models.ZcloudApplication, domainSuffix string) error {
	newapp, err := tp.GenerateAppObject(app.Cluster, app.Namespace, app.TemplateName, domainSuffix)
	if err != nil {
		return err
	}
	inputspec, err := tp.String()
	if err != nil {
		return err
	}
	//check image
	if newapp.Image == "" {
		return fmt.Errorf("the application configure has no image!")
	}
	//check
	if !(newapp.Replicas >= common.ReplicasMin && newapp.Replicas <= common.ReplicasMax) {
		return fmt.Errorf("replcas is not right, its valid range is [%v, %v]!", common.ReplicasMin, common.ReplicasMax)
	}
	if app.Kind != tp.GetAppKind() {
		return fmt.Errorf("the kind of application can not be changed!")
	}
	// compatible for old app with no version
	if app.Version != "" {
		app.Version = newapp.Version
	}
	app.Template = inputspec
	app.Replicas = newapp.Replicas
	app.Image = newapp.Image
	app.InjectServiceMesh = newapp.InjectServiceMesh
	if newapp.DefaultDomainAddr != app.DefaultDomainAddr && newapp.DefaultDomainAddr != "" {
		app.DefaultDomainAddr = newapp.DefaultDomainAddr
	}
	return nil
}

func (tp *NativeAppTemplate) GenerateKubeObject(cluster, namespace, podVersion, domainSuffix string) (map[string]interface{}, error) {
	// translate template to kubernetes resource objects
	objs := make(map[string]interface{})
	switch tp.GetAppKind() {
	case AppKindDeployment:
		deploy := &v1beta1.Deployment{
			TypeMeta:   tp.Deployment.TypeMeta,
			ObjectMeta: tp.Deployment.ObjectMeta,
			Spec:       tp.Deployment.Spec,
		}
		deploy.Name = genAppName(deploy.Name, podVersion)
		deploy.Spec.Template = tp.newPodTemplateSpec(deploy.Spec.Template, podVersion)
		deploy.ObjectMeta = tp.newAppObjectMeta(tp.Deployment.ObjectMeta,
			deploy.Spec.Template.Labels,
			namespace,
			deploy.Name,
			podVersion)
		deploy.Spec.Selector = tp.newAppSelector(deploy.Spec.Selector, deploy.Spec.Template, podVersion)
		objs[AppKindDeployment] = deploy
	default:
		beego.Warn("cant support this application kind:", tp.GetAppKind())
		return nil, fmt.Errorf("cant support this application kind: %s", tp.GetAppKind())
	}
	svcList := []*apiv1.Service{}
	for _, svc := range tp.Services {
		if err := NewKubeSvcValidator(cluster, namespace, tp.GetAppName()).Validator(svc); err != nil {
			return nil, err
		}
		svc.ObjectMeta = tp.newObjectMeta(svc.ObjectMeta, svc.Labels, namespace, svc.Name)
		svcList = append(svcList, svc)
	}
	if len(svcList) > 0 {
		objs[ServiceKind] = svcList
	}
	ingList := []*extensions.Ingress{}
	tp.genDefaultIngressObjects(namespace, tp.GetAppName(), domainSuffix)
	err := error(nil)
	for _, ing := range tp.Ingresses {
		if checkErr := kubeutil.CheckIngressRule(cluster, namespace, ing.Spec.Rules); checkErr != nil {
			// for delete resources
			err = checkErr
		}
		ing.ObjectMeta = tp.newObjectMeta(ing.ObjectMeta, ing.Labels, namespace, ing.Name)
		kubeutil.SetCreatedDefaultAnno(ing)
		ingList = append(ingList, ing)
	}
	if len(svcList) > 0 {
		objs[IngressKind] = ingList
	}
	return objs, err
}

func (tp *NativeAppTemplate) GetAppName() string {
	return tp.Name
}

func (tp *NativeAppTemplate) GetAppKind() string {
	return strings.ToLower(tp.Kind)
}

func (tp *NativeAppTemplate) GetAppVersion() string {
	return tp.Config.Version
}

func (tp *NativeAppTemplate) String() (string, error) {
	ctx, err := json.Marshal(tp)
	if err != nil {
		return "", err
	}

	return string(ctx), nil
}

func (tp *NativeAppTemplate) Image(param []ContainerParam) AppTemplate {
	for _, item := range param {
		podSpec := &apiv1.PodTemplateSpec{}
		switch tp.GetAppKind() {
		case AppKindDeployment:
			podSpec = &tp.Deployment.Spec.Template
		}
		for index, ctn := range podSpec.Spec.Containers {
			if item.Name == ctn.Name {
				podSpec.Spec.Containers[index].Image = item.Image
				break
			}
		}
	}
	return tp
}

func (tp *NativeAppTemplate) Replicas(replicas int) AppTemplate {
	num := int32(replicas)
	switch tp.GetAppKind() {
	case AppKindDeployment:
		tp.Deployment.Spec.Replicas = &num
	}
	return tp
}

func (tp *NativeAppTemplate) DefaultLabel() AppTemplate {
	return tp
}

func (tp *NativeAppTemplate) IsInjectServiceMesh() bool {
	return tp.Config.InjectServiceMesh
}

func (tp *NativeAppTemplate) replaceImagePullAddr(cluster string) AppTemplate {
	initAddr, pullAddr, err := GetClusterHarborAddr(cluster)
	if err != nil {
		beego.Error(err)
		return tp
	}
	if initAddr == "" || pullAddr == "" {
		return tp
	}
	podSpec := &apiv1.PodTemplateSpec{}
	switch tp.GetAppKind() {
	case AppKindDeployment:
		podSpec = &tp.Deployment.Spec.Template
	default:
		return tp
	}
	for i, c := range podSpec.Spec.InitContainers {
		podSpec.Spec.InitContainers[i].Image = strings.Replace(c.Image, initAddr, pullAddr, 1)
	}
	for i, c := range podSpec.Spec.Containers {
		podSpec.Spec.Containers[i].Image = strings.Replace(c.Image, initAddr, pullAddr, 1)
	}
	return tp
}

func (tp *NativeAppTemplate) newPodTemplateSpec(spec apiv1.PodTemplateSpec, podVersion string) apiv1.PodTemplateSpec {
	spec.ObjectMeta = tp.newAppObjectMeta(spec.ObjectMeta,
		spec.Labels,
		spec.ObjectMeta.Namespace,
		spec.ObjectMeta.Name,
		podVersion)
	if tp.Config.ImagePullSecret != "" {
		spec.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{{Name: tp.Config.ImagePullSecret}}
	}
	if tp.IsInjectServiceMesh() {
		spec.Spec.DNSPolicy = apiv1.DNSClusterFirst
	}
	return spec
}

func (tp *NativeAppTemplate) newAppSelector(old *metav1.LabelSelector, podTemplate apiv1.PodTemplateSpec, podversion string) *metav1.LabelSelector {
	selector := old
	if selector == nil {
		selector = &metav1.LabelSelector{}
	}
	if selector.MatchLabels == nil {
		selector.MatchLabels = podTemplate.Labels
	}
	if podversion != "" {
		selector.MatchLabels[keyword.LABEL_PODVERSION_KEY] = podversion
	}
	return selector
}

func (tp *NativeAppTemplate) newObjectMeta(old metav1.ObjectMeta, defLabel map[string]string, namespace, name string) metav1.ObjectMeta {
	meta := old
	meta.Name = name
	meta.Namespace = namespace
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	// set ann
	meta.Annotations[OwnerNameAnnotationKey] = tp.GetAppName()
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	//set labelif
	for k, v := range defLabel {
		meta.Labels[k] = v
	}
	meta.Labels[keyword.LABEL_APPNAME_KEY] = tp.GetAppName()
	return meta
}

func (tp *NativeAppTemplate) newAppObjectMeta(old metav1.ObjectMeta, defLabel map[string]string, namespace, name, podVersion string) metav1.ObjectMeta {
	meta := tp.newObjectMeta(old, defLabel, namespace, name)
	if tp.IsInjectServiceMesh() {
		meta.Annotations[InjectSidecarAnnotationKey] = "true"
	} else {
		meta.Annotations[InjectSidecarAnnotationKey] = "false"
	}
	if tp.GetAppVersion() != "" {
		meta.Labels[keyword.LABEL_APPVERSION_KEY] = tp.GetAppVersion()
	}
	if podVersion != "" {
		meta.Labels[keyword.LABEL_PODVERSION_KEY] = podVersion
	}
	return meta
}

func (tp *NativeAppTemplate) getAppLabelSelector() string {
	labels := make(map[string]string)
	for k, v := range tp.getAppPodLabel() {
		if k != keyword.LABEL_PODVERSION_KEY {
			labels[k] = v
		}
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		beego.Warn("get application label failed:", err)
		return ""
	}
	return selector.String()
}

func (tp *NativeAppTemplate) getAppPodLabel() map[string]string {
	return tp.Deployment.Spec.Template.Labels
}

//generate ingress object
func (tp *NativeAppTemplate) genDefaultIngressObjects(namespace, appname, domainSuffix string) []*extensions.Ingress {
	//var err error
	if domainSuffix == "" {
		return nil
	}
	objectMeta := tp.newObjectMeta(metav1.ObjectMeta{}, nil, namespace, appname)
	objectMeta.Name = GenIngressName(appname)
	domainName := GenerateDomainName(appname)
	var newIng *extensions.Ingress
	for _, svc := range tp.Services {
		defPort := tp.Config.DefaultPort
		if len(svc.Spec.Ports) == 1 {
			defPort = svc.Spec.Ports[0].Port
		}
		ing := GenDefaultIngressObject(svc, objectMeta, defPort, domainName, domainSuffix)
		if ing == nil {
			continue
		}
		// check the host and path whether if existed in app default ingress
		if ingressRuleIsExisted(ing, tp.Ingresses) {
			continue
		}
		if newIng == nil {
			newIng = ing
		} else {
			newIng.Spec.Rules = append(newIng.Spec.Rules, ing.Spec.Rules...)
			newIng.Spec.TLS = append(newIng.Spec.TLS, ing.Spec.TLS...)
		}
	}
	if newIng != nil {
		add := true
		for _, ing := range tp.Ingresses {
			if newIng.Name == ing.Name {
				add = false
				break
			}
		}
		if add {
			tp.Ingresses = append(tp.Ingresses, newIng)
		}
	}
	return tp.Ingresses
}

func ingressRuleIsExisted(defIng *extensions.Ingress, ings []*extensions.Ingress) bool {
	if defIng == nil {
		return false
	}
	// check host
	getSameHostRules := func() (*extensions.IngressRule, *extensions.IngressRule) {
		for _, rule := range defIng.Spec.Rules {
			for _, ing := range ings {
				for _, r := range ing.Spec.Rules {
					if r.Host == rule.Host {
						return &rule, &r
					}
				}
			}
		}
		return nil, nil
	}
	destRule, sameRule := getSameHostRules()
	if destRule == nil || sameRule == nil {
		return false
	}
	if destRule.HTTP == nil || sameRule.HTTP == nil {
		return false
	}
	for _, path1 := range destRule.HTTP.Paths {
		for _, path2 := range destRule.HTTP.Paths {
			if utils.PathsIsEqual(path1.Path, path2.Path) {
				return true
			}
		}
	}
	return false
}

func genAppName(virginName, suffix string) string {
	items := strings.Split(virginName, "-")
	if len(items) > 1 {
		if items[len(items)-1] == suffix {
			return virginName
		}
	}
	return GenerateDeployName(virginName, suffix)
}

func GenDefaultIngressObject(svc *apiv1.Service, ingObjMeta metav1.ObjectMeta, defPort int32, domainName, domainSuffix string) *extensions.Ingress {
	if svc == nil {
		return nil
	}
	if domainSuffix == "" {
		beego.Warn("domain suffix is empty, so cant generate ingress object!")
		return nil
	}
	if len(svc.Spec.Ports) == 0 {
		return nil
	}
	ingress := &extensions.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: IngApiVersion,
		},
	}
	var rules []extensions.IngressRule
	for _, port := range svc.Spec.Ports {
		dPort := port.Port
		// set ingress rule
		ruleValue := extensions.IngressRuleValue{
			HTTP: &extensions.HTTPIngressRuleValue{
				Paths: []extensions.HTTPIngressPath{
					extensions.HTTPIngressPath{
						Backend: extensions.IngressBackend{
							ServiceName: svc.Name,
							ServicePort: intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: dPort,
							},
						},
					},
				},
			},
		}
		hostSuffix := ""
		if defPort != port.Port {
			hostSuffix = strconv.Itoa(int(port.Port))
		}
		rules = append(rules, extensions.IngressRule{
			Host:             GenerateIngressHost(domainName, domainSuffix, hostSuffix),
			IngressRuleValue: ruleValue,
		})
	}
	ingress.ObjectMeta = ingObjMeta
	//ingress.ObjectMeta.ResourceVersion = ingrv
	var specRules []extensions.IngressRule
	// update rule
	for _, rule := range rules {
		ir := -1
		for i, item := range ingress.Spec.Rules {
			if item.Host == rule.Host {
				ir = i
				break
			}
		}
		if ir == -1 {
			// just append a rule
			ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
		} else {
			// update rule: add a path or update path's backend
			// the method may cause some garbage paths
			drule := &ingress.Spec.Rules[ir]
			for _, spath := range rule.HTTP.Paths {
				ipath := -1
				for i, dpath := range drule.HTTP.Paths {
					if dpath.Path == spath.Path {
						ipath = i
					}
				}
				if ipath == -1 {
					// add a path
					drule.HTTP.Paths = append(drule.HTTP.Paths, spath)
				} else {
					// update path's backend
					drule.HTTP.Paths[ipath].Backend = spath.Backend
				}
			}
		}
	}
	// delete ingress rule which service port is not existed in service
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				found := false
				for _, port := range svc.Spec.Ports {
					if path.Backend.ServicePort.IntValue() == int(port.Port) {
						found = true
						break
					}
				}
				if found {
					specRules = append(specRules, rule)
					break
				}
			}
		} else {
			specRules = append(specRules, rule)
		}
	}
	// delete tls host if host is not existed in rule
	var specTLS []extensions.IngressTLS
	for _, item := range ingress.Spec.TLS {
		var hosts []string
		for _, host := range item.Hosts {
			existed := false
			for _, rule := range specRules {
				if rule.Host == host {
					existed = true
					break
				}
			}
			if existed {
				hosts = append(hosts, host)
			}
		}
		if len(hosts) != 0 {
			specTLS = append(specTLS, extensions.IngressTLS{
				Hosts:      hosts,
				SecretName: item.SecretName,
			})
		}
	}

	// set spec rules
	ingress.Spec.Rules = specRules
	ingress.Spec.TLS = specTLS
	// set default annotation
	kubeutil.SetCreatedDefaultAnno(ingress)
	return ingress
}
