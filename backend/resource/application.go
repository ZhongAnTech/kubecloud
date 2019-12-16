package resource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/common"
	"kubecloud/common/keyword"
	"kubecloud/common/utils"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"kubecloud/common/validate"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type ContainerParam struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type RollingUpdateApp struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type AppParam struct {
	Name       string              `json:"name"` //appname
	Containers []ContainerParam    `json:"containers,omitempty"`
	Replicas   *intstr.IntOrString `json:"replicas,omitempty"`
}

type VersionWeight struct {
	Stage   string             `json:"stage"`
	Version string             `json:"version"`
	Weight  intstr.IntOrString `json:"weight"`
}

type AppItem struct {
	models.ZcloudApplication
	ReplicasConstrast string          `json:"replicas_constrast"`
	Status            string          `json:"status"`
	VersionList       []VersionWeight `json:"version_list"`
	CreateAt          string          `json:"create_at"`
	UpdateAt          string          `json:"update_at"`
}

type AppPod struct {
	Pod    `json:",inline"`
	Weight int `json:"weight"`
}

type AppDetail struct {
	AppItem
	Ingress []SimpleIngressDetail `json:"ingress,omitempty"`
	Service ServiceDetail         `json:"service,omitempty"`
	Pods    []*AppPod             `json:"pods,omitempty"`
}

type AppRes struct {
	Cluster      string
	DomainSuffix string
	Client       kubernetes.Interface
	Appmodel     *dao.AppModel
	SvcRes       *ServiceRes
	IngRes       *IngressRes
	versionModel *dao.VersionModel
	listNSFunc   NamespaceListFunction
}

type AppPodBasicParam struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	PodName   string `json:"pod_name"`
}

type AppPodDetail struct {
	models.ZcloudApplication `json:",inline"`
	Pod                      AppPod `json:"pod"`
}

func NewAppRes(cluster string, get NamespaceListFunction) (*AppRes, error) {
	if cluster == "" {
		return &AppRes{
			Cluster:      cluster,
			Appmodel:     dao.NewAppModel(),
			versionModel: dao.NewVersionModel(),
			listNSFunc:   get,
		}, nil
	}
	clusterInfo, err := dao.GetCluster(cluster)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	client, err := service.GetClientset(cluster)
	if err != nil {
		if cluster != "" {
			return nil, common.NewInternalServerError().SetCause(err)
		}
	}

	svc := NewServiceRes(cluster, nil)
	ing, _ := NewIngressRes(cluster, client, nil)
	return &AppRes{
		Cluster:      cluster,
		DomainSuffix: clusterInfo.DomainSuffix,
		Appmodel:     dao.NewAppModel(),
		versionModel: dao.NewVersionModel(),
		listNSFunc:   get,
		Client:       client,
		SvcRes:       svc,
		IngRes:       ing,
	}, nil
}

func (ar *AppRes) GetAppList(namespace string, defFilter *utils.DefaultFilter, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	appList := []AppItem{}
	nslist := []string{}
	if namespace != common.AllNamespace {
		nslist = append(nslist, namespace)
	} else {
		nslist = ar.listNSFunc()
	}
	if len(nslist) == 0 {
		beego.Warn("no namespace for user to view!")
		return utils.InitQueryResult(appList, filterQuery), nil
	}
	defFilter.AppendFilter("namespace", nslist, utils.FilterOperatorIn)
	res, err := ar.Appmodel.GetAppList(defFilter, filterQuery)
	if err != nil {
		return nil, err
	}
	list, ok := res.List.([]models.ZcloudApplication)
	if !ok {
		return nil, fmt.Errorf("data type is not right!")
	}
	for _, item := range list {
		aitem := AppItem{}
		aitem.ZcloudApplication = item
		aitem.Template = ""
		aitem.CreateAt = item.CreateAt.Format("2006-01-02 15:04:05")
		aitem.UpdateAt = item.UpdateAt.Format("2006-01-02 15:04:05")
		aitem.ReplicasConstrast, aitem.Status = ar.GetPods(item)
		appList = append(appList, aitem)
	}
	res.List = appList

	return res, nil
}

func (ar *AppRes) GetPods(app models.ZcloudApplication) (string, string) {
	expectReplicas := app.StatusReplicas
	if expectReplicas == 0 {
		expectReplicas = int32(app.Replicas)
	}
	pods := fmt.Sprintf("%v / %v", app.ReadyReplicas, expectReplicas)
	status := "NotReady"
	if app.ReadyReplicas == expectReplicas {
		status = "Running"
	} else if app.ReadyReplicas != 0 {
		status = "Warning"
	}
	return pods, status
}

func (ar *AppRes) GetAppDetail(namespace, name string) (*AppDetail, error) {
	app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, name)
	if err != nil {
		return nil, err
	}

	detail := AppDetail{}
	detail.ZcloudApplication = *app
	detail.ReplicasConstrast, detail.Status = ar.GetPods(*app)
	detail.CreateAt = app.CreateAt.Format("2006-01-02 15:04:05")
	detail.UpdateAt = app.UpdateAt.Format("2006-01-02 15:04:05")
	if tpl, err := CreateAppTemplateByApp(detail.ZcloudApplication); err == nil {
		if template, err := AppTemplateToYamlString(tpl, app.Cluster, app.Namespace, app.PodVersion, ar.DomainSuffix); err == nil {
			detail.Template = template
		} else {
			beego.Warn(err)
		}
	} else {
		beego.Warn(err)
	}
	vs, _ := ar.versionModel.GetVersionList(ar.Cluster, namespace, name)
	for _, item := range vs {
		detail.VersionList = append(detail.VersionList, VersionWeight{
			Stage:   item.Stage,
			Version: item.Version,
			Weight:  intstr.FromInt(item.Weight),
		})
	}
	if detail.Ingress, err = ar.IngRes.GetSimpleIngressDetail(namespace, name); err != nil {
		beego.Warn("get ingress detail failed:", err)
	}
	// Pods
	detail.Pods, err = ar.getAppPodList(app, vs)
	if err != nil {
		beego.Error("Get Pods information failed: " + err.Error())
		return nil, err
	}
	if svc, err := ar.SvcRes.GetServiceDetail(namespace, name, getBestNodeIP(detail.Pods)); err != nil {
		beego.Warn("get service detail failed:", err)
	} else {
		detail.Service = svc
	}
	return &detail, nil
}

func (ar *AppRes) GetAppPodStatus(namespace, appName, podName string) (interface{}, error) {
	_, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appName)
	if err != nil {
		return "", err
	}
	pod, err := ar.Client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Status, nil
}

func (ar *AppRes) GetAppPodLog(namespace, appName, podName, containerName string) (string, error) {
	_, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appName)
	if err != nil {
		return "", err
	}

	tailLines := int64(1000)
	body, err := ar.Client.CoreV1().Pods(namespace).GetLogs(podName, &apiv1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}).Do().Raw()

	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (ar *AppRes) InstallApp(projectid int64,
	namespace, tname string,
	template Template,
	eparam *ExtensionParam) error {
	if err := KubeNamespaceCreate(ar.Client, ar.Cluster, namespace); err != nil {
		return err
	}
	CreateHarborSecret(ar.Cluster, namespace)
	if err := template.Validate(); err != nil {
		return common.NewBadRequest().SetCause(err)
	}
	if err := template.Default(ar.Cluster).Deploy(projectid, ar.Cluster, namespace, tname, eparam); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	return nil
}

func (ar *AppRes) UninstallApp(app models.ZcloudApplication) error {
	if app.Template == "" {
		return nil
	}
	template, err := CreateAppTemplateByApp(app)
	if err != nil {
		return err
	}
	kr := NewKubeAppRes(ar.Client, ar.Cluster, app.Namespace, ar.DomainSuffix, app.Kind)

	return kr.DeleteAppResource(template, app.PodVersion)
}

func (ar *AppRes) DeleteApp(namespace, appname string) error {
	app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
	if err != nil {
		if err == orm.ErrNoRows {
			return nil
		} else {
			return err
		}
	}
	err = ar.UninstallApp(*app)
	if err != nil {
		return err
	}
	err = ar.Appmodel.DeleteApp(*app)
	if err == nil {
		// delete version info
		if err = ar.versionModel.DeleteAllVersion(ar.Cluster, namespace, appname); err != nil {
			beego.Warn("delete application version failed: "+err.Error(),
				"namespace: "+namespace, "appname: "+appname)
		}
	}
	return err
}

func (ar *AppRes) Restart(namespace, appname string) error {
	app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
	if err != nil {
		if err == orm.ErrNoRows {
			beego.Warn(ar.Cluster, namespace, appname, " is not existed")
			return nil
		}
		return err
	}
	template, err := CreateAppTemplateByApp(*app)
	if err != nil {
		return err
	}
	return NewKubeAppRes(ar.Client, ar.Cluster, namespace, ar.DomainSuffix, app.Kind).Restart(app, template)
}

func (ar *AppRes) ReconfigureApp(app models.ZcloudApplication, template AppTemplate) (*AppDetail, error) {
	kr := NewKubeAppRes(ar.Client, ar.Cluster, app.Namespace, ar.DomainSuffix, app.Kind)
	exist, err := kr.CheckAppIsExisted(app.Name, app.PodVersion)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	if exist {
		oldTpl, err := CreateAppTemplateByApp(app)
		if err != nil {
			return nil, common.NewInternalServerError().SetCause(err)
		}
		//update
		err = kr.UpdateAppResource(&app, template, oldTpl, true)
		if err != nil {
			return nil, common.NewInternalServerError().SetCause(err)
		}
		beego.Warn("the app is reconfigured,", "cluster: "+ar.Cluster, "namespace: "+app.Namespace, "appname: "+app.Name)
	} else {
		if err := template.UpdateAppObject(&app, ar.DomainSuffix); err != nil {
			return nil, common.NewBadRequest().SetCause(err)
		}
		//recreate
		err = kr.CreateAppResource(template, app.PodVersion)
		if err != nil {
			return nil, common.NewInternalServerError().SetCause(err)
		}
		beego.Warn("the app is recreated,", "cluster: "+ar.Cluster, "namespace: "+app.Namespace, "appname: "+app.Name)
	}
	// update app info
	err = ar.Appmodel.UpdateApp(&app, true)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	appDetail, err := ar.GetAppDetail(app.Namespace, app.Name)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}

	return appDetail, nil
}

func (ar *AppRes) RollingUpdateApp(namespace, appname string, param []ContainerParam) error {
	app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
	if err != nil {
		if err == orm.ErrNoRows {
			return common.NewNotFound().SetCause(err)
		} else if err == orm.ErrMultiRows {
			return common.NewConflict().SetCause(err)
		} else {
			return common.NewInternalServerError().SetCause(err)
		}
	}
	template, err := CreateAppTemplateByApp(*app)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	kr := NewKubeAppRes(ar.Client, ar.Cluster, namespace, ar.DomainSuffix, app.Kind)
	if err = kr.UpdateAppResource(app, template.Image(param), nil, false); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	beego.Debug(fmt.Sprintf("new image for %s/%s/%s is %s!", ar.Cluster, namespace, appname, app.Image))
	if err = ar.Appmodel.UpdateApp(app, true); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	return nil
}

func (ar *AppRes) BatchRollingUpdateApp(namespace string, apps []RollingUpdateApp) *common.Error {
	chunkSize := 10

	validateFunc := func(data interface{}) interface{} {
		app := data.(RollingUpdateApp)
		// find app
		_, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, app.Name)
		if err != nil {
			if err == orm.ErrNoRows {
				return common.NewNotFound().SetCause(fmt.Errorf(`Application "%s" not found!`, app.Name))
			} else if err == orm.ErrMultiRows {
				return common.NewConflict().SetCause(fmt.Errorf(`Application "%s" not unique!`, app.Name))
			} else {
				return common.NewInternalServerError().SetCause(fmt.Errorf(`Update application failed: %v`, err))
			}
		}
		// check image url
		if err := HarborEnsureImageUrl(app.Image); err != nil {
			return err
		}
		return nil
	}

	updateFunc := func(data interface{}) interface{} {
		app := data.(RollingUpdateApp)
		param := []ContainerParam{
			ContainerParam{Name: app.Name, Image: app.Image},
		}
		err := ar.RollingUpdateApp(namespace, app.Name, param)
		if err != nil {
			beego.Error("BatchRollingUpdateApp failed", "app:", app, err)
			return err
		}
		return nil
	}

	slice := utils.ToInterfaceSlice(apps)
	if err := utils.GoThrough(slice, validateFunc, chunkSize); err != nil {
		ret, ok := err.(*common.Error)
		if ok {
			return ret
		}
		return common.NewInternalServerError().SetCause(ret)
	}
	if err := utils.GoThrough(slice, updateFunc, chunkSize); err != nil {
		ret, ok := err.(*common.Error)
		if ok {
			return ret
		}
		return common.NewInternalServerError().SetCause(ret)
	}
	return nil
}

func (ar *AppRes) ScaleApp(namespace, appname string, replicas int) error {
	item, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
	if err != nil {
		return err
	}
	template, err := CreateAppTemplateByApp(*item)
	if err != nil {
		return err
	}
	kr := NewKubeAppRes(ar.Client, ar.Cluster, namespace, ar.DomainSuffix, item.Kind)
	if err := kr.Scale(item, template, replicas); err != nil {
		return err
	}
	tplStr, err := template.Replicas(replicas).String()
	if err != nil {
		return err
	}
	item.Replicas = replicas
	item.Template = tplStr
	return ar.Appmodel.UpdateApp(item, true)
}

func (ar *AppRes) SetVersion(app *models.ZcloudApplication, stage string, weight, currep int) error {
	// create app version
	version := &models.ZcloudVersion{
		Cluster:   app.Cluster,
		Namespace: app.Namespace,
		// name is application name
		Name:       app.Name,
		Kind:       app.Kind,
		Version:    GetResourceVersion(app, ResTypeApp, ""),
		PodVersion: app.PodVersion, //maybe empty
		Weight:     weight,
		// new or normal
		Stage:             stage,
		Replicas:          app.Replicas,
		CurReplicas:       currep,
		TemplateName:      app.TemplateName,
		Image:             app.Image,
		Template:          app.Template,
		InjectServiceMesh: app.InjectServiceMesh,
	}
	return ar.versionModel.SetVersionWeight(version)
}

func (ar *AppRes) SetDeployStatus(namespace, appname, status string) error {
	return ar.Appmodel.SetDeployStatus(ar.Cluster, namespace, appname, status)
}

func (ar *AppRes) getAppPodList(app *models.ZcloudApplication, vs []models.ZcloudVersion) ([]*AppPod, error) {
	podSelector := keyword.LABEL_APPNAME_KEY + "=" + app.Name
	podList, err := GetPods(ar.Cluster, app.Namespace, podSelector, app.Replicas)
	if err != nil {
		beego.Error("Get Pods information failed: " + err.Error())
		return nil, err
	}
	// has dynamic weight
	hasVerList := (len(vs) != 0)
	podsNumMap, otherPodNum := getRunningPodNumMap(hasVerList, podList)
	weightMap, otherWeight := getServiceWeightMap(vs)
	for _, v := range vs {
		if podsNumMap[v.PodVersion] != 0 {
			continue
		}
		for _, item := range vs {
			if item.Version == v.Version {
				continue
			}
			// set weight to 0 which verison's pods are not running
			// and add this weight to other version
			weightMap[item.PodVersion] += weightMap[v.PodVersion]
			weightMap[v.PodVersion] = 0
		}
	}
	appPodList := []*AppPod{}
	for _, item := range podList {
		pod := AppPod{
			Weight: 0,
		}
		pod.Pod = *item
		averWeight := models.DEFAULT_WEIGHT
		svcWeight := otherWeight
		if w, ok := weightMap[pod.Version]; ok {
			svcWeight = w
		}
		podsNum := otherPodNum
		if num, ok := podsNumMap[pod.Version]; ok {
			podsNum = num
			if podsNum != 0 {
				averWeight = svcWeight / podsNum
			}
		} else {
			if podsNum != 0 {
				averWeight = svcWeight / podsNum
			}
		}
		if pod.Status == string(apiv1.PodRunning) {
			pod.Weight = averWeight
		}
		appPodList = append(appPodList, &pod)
	}
	// switch pod version to app version
	verList := vs
	if !hasVerList {
		verList = []models.ZcloudVersion{models.ZcloudVersion{Version: app.Version, PodVersion: app.PodVersion}}
	}
	for _, v := range verList {
		for i, pod := range appPodList {
			if pod.Version == v.PodVersion {
				appPodList[i].Version = v.Version
			}
		}
	}

	return appPodList, nil
}

func (ar *AppRes) SetLabels(namespace, name string, labels map[string]string) error {
	app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, name)
	if err != nil {
		if err == orm.ErrNoRows {
			beego.Warn(fmt.Sprintf("application(%s/%s/%s) is not existed!", ar.Cluster, namespace, name))
			return nil
		}
		return err
	}
	if err := validate.ValidateLabels(keyword.K8S_RESOURCE_TYPE_APP, labels); err != nil {
		return err
	}
	labelStr, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	if string(labelStr) != app.Labels {
		return ar.Appmodel.SetLabels(ar.Cluster, namespace, name, string(labelStr))
	}
	return nil
}

func CreateHarborSecret(cluster, namespace string) error {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Warning(fmt.Sprintf("create harbor secret failed: %v", err.Error()))
		return err
	}
	clusterInfo, err := dao.GetCluster(cluster)
	if err != nil {
		beego.Warning(fmt.Sprintf("create harbor secret failed: %v", err.Error()))
		return err
	}
	harbor, err := dao.GetHarbor(clusterInfo.Registry)
	if err != nil {
		beego.Warning(fmt.Sprintf("create harbor secret failed: %v", err.Error()))
		return err
	}
	harborSecretName := fmt.Sprintf("harbor-%v", harbor.HarborName)
	harborInfo := make(map[string]interface{})
	harborInfo[harbor.HarborAddr] = map[string]string{
		"username": harbor.HarborUser,
		"password": harbor.HarborPassword,
		"auth":     harbor.HarborAuth,
	}
	if clusterInfo.ImagePullAddr != "" && clusterInfo.ImagePullAddr != harbor.HarborAddr {
		harborInfo[clusterInfo.ImagePullAddr] = map[string]string{
			"username": harbor.HarborUser,
			"password": harbor.HarborPassword,
			"auth":     harbor.HarborAuth,
		}
	}
	auth, _ := json.Marshal(harborInfo)
	harborSec, err := client.CoreV1().Secrets(namespace).Get(harborSecretName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err = client.CoreV1().Secrets(namespace).Create(&apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      harborSecretName,
			},
			Type: apiv1.SecretTypeDockercfg,
			Data: map[string][]byte{
				".dockercfg": auth,
			},
		})
	} else {
		if string(harborSec.Data[".dockercfg"]) == string(auth) {
			return nil
		}
		harborSec.Data = map[string][]byte{".dockercfg": auth}
		_, err = client.CoreV1().Secrets(namespace).Update(harborSec)
	}
	if err != nil {
		beego.Warning(fmt.Sprintf("set harbor secret failed: %v", err.Error()))
	}
	return err
}

func getKubeResNumber(res string) (int64, error) {
	bind := map[string]int64{"ki": 1 / (2 ^ 10), "mi": 1, "gi": (2 ^ 10), "ti": (2 ^ 20), "pi": (2 ^ 30), "ei": (2 ^ 40)}
	ints := map[string]int64{"k": 1 / (10 ^ 3), "m": 1, "g": (10 ^ 3), "t": (10 ^ 6), "p": (10 ^ 9), "e": (10 ^ 12)}
	//default g
	dest := strings.TrimSpace(strings.ToLower(res))
	for key, value := range bind {
		if strings.HasSuffix(dest, key) {
			nb, err := strconv.Atoi((strings.TrimRight(dest, key)))
			if err != nil {
				return 0, err
			}
			return int64(nb) * value, nil
		}
	}
	for key, value := range ints {
		if strings.HasSuffix(dest, key) {
			nb, err := strconv.Atoi((strings.TrimRight(dest, key)))
			if err != nil {
				return 0, err
			}
			return int64(nb) * value, nil
		}
	}
	nb, err := strconv.Atoi(dest)
	if err != nil {
		return 0, err
	}
	return int64(nb) * (10 ^ 3), nil
}

func getAccurateAppSuffix(app models.ZcloudApplication, v models.ZcloudVersion) string {
	if v.Stage == models.STAGE_NORMAL && v.PodVersion != app.PodVersion {
		return app.PodVersion
	}
	return v.PodVersion
}

func getBestNodeIP(pods []*AppPod) string {
	if len(pods) > 0 {
		return pods[0].NodeIP
	}
	return ""
}

type AppEvent struct {
	EventLevel   string `json:"event_level"`
	EventObject  string `json:"event_object"`
	EventType    string `json:"event_type"`
	EventMessage string `json:"event_message"`
	EventTime    string `json:"event_time"`
}

func (ar *AppRes) GetAppEvent(namespace, appName string) ([]*models.ZcloudEvent, error) {
	eventList, err := dao.GetAppEvents(ar.Cluster, namespace, appName)
	if err != nil {
		return nil, err
	}
	return eventList, nil
}

func getServiceWeightMap(verList []models.ZcloudVersion) (map[string]int, int) {
	weightMap := make(map[string]int)
	otherWeight := models.MAX_WEIGHT
	for _, vw := range verList {
		weightMap[vw.PodVersion] = vw.Weight
		otherWeight -= vw.Weight
		if otherWeight < 0 {
			otherWeight = models.MIN_WEIGHT
		}
	}
	return weightMap, otherWeight
}

func getRunningPodNumMap(hasVerList bool, pods []*Pod) (map[string]int, int) {
	// pvn: pod:version-num
	pvnMap := make(map[string]int)
	otherPodNum := 0
	// calc pods num for given pod version
	for _, pod := range pods {
		if pod.Status == string(apiv1.PodRunning) {
			if pod.Version != "" && hasVerList {
				pvnMap[pod.Version]++
			} else {
				otherPodNum++
			}
		}
	}
	return pvnMap, otherPodNum
}
