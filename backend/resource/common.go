package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
	app "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SidecarStatusAnnotation  = "sidecar.istio.io/status"
	SidecarEnvoyVolume       = "istio-envoy"
	SidecarCertVolume        = "istio-certs"
	SidecarContainerName     = "istio-proxy"
	SidecarInitContainerName = "istio-init"
)

const (
	AppKindDaemonSet              = "daemonset"
	AppKindDeployment             = "deployment"
	ServiceKind                   = "service"
	IngressKind                   = "ingress"
	ConfigMapKind                 = "configmap"
	SecretKind                    = "secret"
	BasenameAnnotationKey         = "basename"
	DomainNameAnnotationKey       = "domain_name"
	TemplateNameAnnotationKey     = "template_name"
	DescriptionAnnotationKey      = "description"
	OwnerNameAnnotationKey        = "owner_name"
	InjectSidecarAnnotationKey    = "sidecar.istio.io/inject"
	InjectSidecarAnnotationStatus = "sidecar.istio.io/status"
	IngressWeightAnnotationKeyPre = "traefik.ingress.kubernetes.io.weight/"
	ENABLED_KUBE_MONKEY           = "kube-monkey/enabled"
	DEFAULT_PROJECT_ID            = 0
	YamlSeparator                 = "---\n"
	AppAPIVersion                 = "apps/v1beta1"
	SvcApiVersion                 = "v1"
	IngApiVersion                 = "extensions/v1beta1"

	AnnotationKubernetesIngressClass = "kubernetes.io/ingress.class"
	DefaultIngressClass              = "traefik"
)

const (
	OWNER_KIND_APPLICATION = "application"
	OWNER_KIND_JOB         = "job"
	SYNC_TIMEOUT           = 500 //ms
	SYNC_CHECK_STEP        = 20  //ms

	ResTypeApp      ResType = "app"
	ResTypePod      ResType = "pod"
	ResTypeDeploy   ResType = "deploy"
	ResTypeTemplate ResType = "template"
	ResTypeImage    ResType = "image"
)

const (
	NotifyCmdbTypeApp  = "app"
	NotifyCmdbTypeNode = "node"
)

var (
	cmdbServer   = beego.AppConfig.String("cmdb::cmdbServer")
	cmdbPlatform = beego.AppConfig.String("cmdb::platform")
)

var ReserveContainerNames = []string{SidecarContainerName, SidecarInitContainerName}
var ReserveVolumes = []string{SidecarEnvoyVolume, SidecarCertVolume}

type NamespaceListFunction func() []string
type CheckPermFunction func(string) bool
type PatcherFunction func(app models.ZcloudApplication)
type ResType string

// func GetClient(cluster string) (kubernetes.Interface, error) {
// 	return service.GetClientset(cluster)
// }

func GetListOption(key, value string) (metav1.ListOptions, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{key: value},
	})
	if err != nil {
		return metav1.ListOptions{}, err
	}
	return metav1.ListOptions{LabelSelector: selector.String()}, nil
}

func GetContainerVersion(image string) string {
	var dstr string
	path := strings.Split(image, "/")
	if len(path) > 1 {
		dstr = path[len(path)-1]
	} else {
		dstr = path[0]
	}
	items := strings.Split(dstr, ":")
	if len(items) > 1 {
		return items[len(items)-1]
	} else {
		return "latest"
	}
}

func GetResourceVersion(res interface{}, resType ResType, exparam string) string {
	version := DefaultVersion
	typeIsRight := false
	switch resType {
	case ResTypeApp:
		if app, ok := res.(*models.ZcloudApplication); ok {
			typeIsRight = true
			version = app.Version
			if version == "" {
				version = GetImageVersion(app.Image)
			}
		}
	case ResTypePod:
		if pod, ok := res.(*apiv1.Pod); ok {
			typeIsRight = true
			if v, ok := pod.Labels[keyword.LABEL_PODVERSION_KEY]; ok {
				version = v
			} else {
				appname := pod.Labels[keyword.LABEL_APPNAME_KEY]
				for _, container := range pod.Spec.Containers {
					if container.Name == appname {
						version = GetImageVersion(container.Image)
						break
					}
				}
			}
		}
	case ResTypeDeploy:
		if deploy, ok := res.(*app.Deployment); ok {
			typeIsRight = true
			if v, ok := deploy.Labels[keyword.LABEL_APPVERSION_KEY]; ok {
				// firstly get app version
				version = v
			} else {
				if v, ok := deploy.Labels[keyword.LABEL_PODVERSION_KEY]; ok {
					// secondly get pod version
					version = v
				} else {
					// exparam is app.Image
					version = GetImageVersion(exparam)
				}
			}
		}
	case ResTypeImage:
		version = GetImageVersion(res.(string))
	}
	if !typeIsRight {
		beego.Warn(fmt.Sprintf("res real type is not %s, please check", resType))
	}
	return version
}

func GenServiceName(templateKind, svcname string) string {
	return svcname
}

func GenHeadlessSvcName(templateKind, svcname string) string {
	return svcname
}

func GenIngressName(appname string) string {
	return fmt.Sprintf("ing-%s", appname)
}

func GenResourceQuotaName(namespace string) string {
	return fmt.Sprintf("resquota-%s", namespace)
}

func GenerateDeployName(appname, suffix string) string {
	dpname := appname
	if suffix != "" {
		dpname += "-" + suffix
	}
	return dpname
}

func GenerateIngressHost(domainName, domainSuffix, port string) string {
	if port != "" {
		return strings.ToLower(fmt.Sprintf("%s-%s.%s", domainName, port, domainSuffix))
	} else {
		return strings.ToLower(fmt.Sprintf("%s.%s", domainName, domainSuffix))
	}
}

func GenerateDomainName(svcname string) string {
	return strings.Replace(svcname, "_", "-", -1)
}

func GetApplicationNameBySvcName(svcname string) string {
	if strings.HasPrefix(svcname, "svc-") {
		return strings.TrimPrefix(svcname, "svc-")
	} else if strings.HasPrefix(svcname, "hlsvc-") {
		return strings.TrimPrefix(svcname, "hlsvc-")
	}
	return svcname
}

func GetClusterHarborAddr(cluster string) (string, string, error) {
	detail, err := dao.GetCluster(cluster)
	if err != nil {
		return "", "", fmt.Errorf("get cluster detail information by cluster name failed:" + err.Error())
	}
	registry, err := dao.GetHarbor(detail.Registry)
	if err != nil {
		return "", "", fmt.Errorf("get harbor detail information by harbor name failed:" + err.Error())
	}
	return registry.HarborAddr, detail.ImagePullAddr, nil
}

func CheckImageValidate(param interface{}) error {
	imageIsValidate := func(image string) bool {
		imageParts := strings.Split(image, "/")
		imageName := imageParts[len(imageParts)-1]
		items := strings.Split(imageName, ":")

		if len(items) == 2 {
			return len(strings.Replace(items[1], " ", "", -1)) > 0
		}

		return false
	}
	if cparam, ok := param.([]ContainerParam); ok {
		for _, ctn := range cparam {
			if !imageIsValidate(ctn.Image) {
				return common.NewBadRequest().SetCause(fmt.Errorf(`invalid image "%v"`, ctn.Image))
			}
		}
	}
	return nil
}

func GetImageVersion(image string) string {
	v := DefaultVersion
	path := strings.Split(image, "/")
	if len(path) > 1 {
		items := strings.Split(path[len(path)-1], ":")
		if len(items) > 1 {
			v = items[len(items)-1]
		}
	}

	return v
}

// wait some obj for synchronizing to database
func WaitSync(param interface{}, step, ot time.Duration, check func(interface{}) error) error {
	t := time.NewTicker(step * time.Millisecond)
	tmout := time.After(ot * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := check(param); err == nil {
				return nil
			} else {
				beego.Error(err)
			}
		case <-tmout:
			return nil //fmt.Errorf("synchronize ingress information timeout, please refresh yourself")
		}
	}
}

func AddClusterK8sConfig(cluster, certificate string) error {
	configPath := beego.AppConfig.String("k8s::configPath")
	err := os.MkdirAll(configPath, 0766)
	if err != nil {
		return fmt.Errorf("init K8S configure failed: directory is not existed, %s", err.Error())
	}
	fileObj, err := os.OpenFile(configPath+"/"+cluster, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("init K8S configure failed: create configure file failed, %s", err.Error())
	}
	if _, err := io.WriteString(fileObj, certificate); err != nil {
		return fmt.Errorf("init K8S configure failed: write configure file failed, %s", err.Error())
	}
	return nil
}

func MakeK8sConfig() {
	configPath := beego.AppConfig.String("k8s::configPath")
	err := os.MkdirAll(configPath, 0766)
	if err != nil {
		beego.Error(fmt.Sprintf("Failed to make the k8sconfig dir: %v", err.Error()))
		os.Exit(2)
	}
	clusters, err := dao.GetAllClusters()
	if err != nil {
		beego.Error(fmt.Sprintf("Failed to get cluster list: %v", err.Error()))
		os.Exit(2)
	}
	for _, icluster := range clusters {
		fileObj, err := os.OpenFile(configPath+"/"+icluster.ClusterId, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			beego.Error(fmt.Sprintf("Failed to open the file: %v", err.Error()))
			os.Exit(2)
		}
		if _, err := io.WriteString(fileObj, icluster.Certificate); err != nil {
			beego.Error(fmt.Sprintf("init K8S cluster %v configure failed: %v", icluster.ClusterId, err.Error()))
		}
		beego.Info(fmt.Sprintf("init K8S cluster %v configure successfully!", icluster.ClusterId))
	}
}

func InitK8sConfig() {
	MakeK8sConfig()
	go CheckClusterApi()
}

func SentRequestToCmdb(method, urlStr string, body io.Reader) ([]byte, error) {
	rep, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	rep.Header.Set("Content-Type", "application/json")
	rep.Header.Set("api", "pass")
	resp, err := utils.HttpClient.Do(rep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	} else {
		return nil, fmt.Errorf("%s", respBody)
	}
}

func init() {
	if cmdbServer != "" {
		if cmdbPlatform != "za-boom3" || cmdbPlatform != "zatech-boom3" {
			beego.Warn("please set cmdb platform 'zatech-boom3' or 'za-boom3'")
			cmdbServer = ""
		}
	}
}
