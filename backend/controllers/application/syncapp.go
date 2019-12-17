package application

import (
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego"
	v1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubecloud/backend/controllers/util"
	"kubecloud/backend/dao"
	"kubecloud/backend/resource"
	"kubecloud/common/keyword"
)

type appStatus struct {
	ReadyReplicas     int32
	AvailableReplicas int32
	AvailableStatus   string
	Message           string
}

type syncApplication struct {
	appHandler *dao.AppModel
	cluster    string
}

func newSyncApplication(cluster string) *syncApplication {
	return &syncApplication{
		appHandler: dao.NewAppModel(),
		cluster:    cluster,
	}
}

// update if the app is existed, or add it
func (sa *syncApplication) syncDeployApplication(deployment v1beta1.Deployment) error {
	appname := getAppNameByDeploy(deployment)
	if !sa.appHandler.AppExist(sa.cluster, deployment.Namespace, appname) {
		return fmt.Errorf("application(%s/%s/%s) is not existed in db, the deployment is %s!", sa.cluster, deployment.Namespace, appname, deployment.Name)
	} else {
		return sa.updateDeployStatus(appname, deployment)
	}
}

func (sa *syncApplication) updateDeployStatus(appname string, deployment v1beta1.Deployment) error {
	app, err := sa.appHandler.GetAppByName(sa.cluster, deployment.Namespace, appname)
	if err != nil {
		return err
	}
	if deployment.Labels["heritage"] != "Tiller" {
		version := resource.GetResourceVersion(&deployment, resource.ResTypeDeploy, app.Image)
		if resource.GetResourceVersion(app, resource.ResTypeApp, "") != version {
			beego.Warn(fmt.Sprintf("application(%s/%s/%s) dont need update for versions(%s/%s) are not equal!", app.Cluster, app.Namespace, app.Name, version, resource.GetResourceVersion(app, resource.ResTypeApp, "")))
			return nil
		}
	}

	needUpdate := false
	smFlag := util.GetAnnotationStringValue(resource.InjectSidecarAnnotationKey, deployment.Annotations, "")

	if deployment.Labels["heritage"] == "Tiller" {
		deployment.TypeMeta = metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1beta1",
		}
		deployment.ObjectMeta.ResourceVersion = ""
		deployment.ObjectMeta.SelfLink = ""
		deployment.ObjectMeta.UID = ""
		deployment.ObjectMeta.Generation = 0
		deployment.ObjectMeta.CreationTimestamp = metav1.Time{}
		nativeTemplate := resource.NativeAppTemplate{
			TypeMeta:   deployment.TypeMeta,
			ObjectMeta: deployment.ObjectMeta,
			Deployment: &deployment,
		}
		jsonData, err := json.Marshal(nativeTemplate)
		if err != nil {
			return err
		}

		app.Template = string(jsonData)
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: deployment.Spec.Selector.MatchLabels,
		})
		if err != nil {
			return err
		}
		app.LabelSelector = selector.String()
		needUpdate = true
	}

	// because app.InjectServiceMesh "" is equal with "false"
	if (app.InjectServiceMesh == "" || app.InjectServiceMesh == "false") && smFlag == "true" ||
		(smFlag == "false" && app.InjectServiceMesh == "true") {
		app.InjectServiceMesh = smFlag
		needUpdate = true
	}
	if app.Replicas != int(*deployment.Spec.Replicas) {
		app.Replicas = int(*deployment.Spec.Replicas)
		template, err := resource.CreateAppTemplateByApp(*app)
		if err != nil {
			beego.Error("create app template by app failed:", err)
		} else {
			//synchronize information
			if tstr, err := template.Replicas(app.Replicas).String(); err == nil {
				app.Template = tstr
				needUpdate = true
			} else {
				beego.Error("template switch to string failed:", err)
			}
		}
	}
	if app.StatusReplicas != deployment.Status.Replicas {
		app.StatusReplicas = deployment.Status.Replicas
		needUpdate = true
	}
	if app.ReadyReplicas != deployment.Status.ReadyReplicas {
		app.ReadyReplicas = deployment.Status.ReadyReplicas
		needUpdate = true
	}
	if app.AvailableReplicas != deployment.Status.AvailableReplicas {
		app.AvailableReplicas = deployment.Status.AvailableReplicas
		needUpdate = true
	}
	if app.UpdatedReplicas != deployment.Status.UpdatedReplicas {
		app.UpdatedReplicas = deployment.Status.UpdatedReplicas
		needUpdate = true
	}
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == v1beta1.DeploymentAvailable {
			if string(condition.Status) != app.AvailableStatus {
				app.AvailableStatus = string(condition.Status)
				needUpdate = true
			}
			if condition.Message != app.Message {
				app.Message = condition.Message
				needUpdate = true
			}
			break
		}
	}
	if podv, ok := deployment.Labels[keyword.LABEL_PODVERSION_KEY]; !ok || podv == "" {
		app.PodVersion = ""
		needUpdate = true
	}
	if needUpdate {
		err = sa.appHandler.UpdateApp(app, false)
		if err != nil {
			beego.Error("Update application", sa.cluster, app.Namespace, app.Name, "failed for", err)
		} else {
			beego.Info("Update application", sa.cluster, app.Namespace, app.Name, "successfully")
		}
		return err
	}
	return nil
}

func getAppNameByDeploy(deploy v1beta1.Deployment) string {
	appname := deploy.Name
	if deploy.Labels["heritage"] != "Tiller" {
		if v, ok := deploy.Labels[keyword.LABEL_APPNAME_KEY]; ok {
			appname = v
		}
	}
	return appname
}
