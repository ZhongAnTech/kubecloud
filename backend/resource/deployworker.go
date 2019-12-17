package resource

import (
	"fmt"
	"kubecloud/common/utils"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type ExtensionParam struct {
	Force   bool //when user deploy its app and the app is existed in other namespace, the old app will be deleted
	Patcher PatcherFunction
}

type DeployWorker struct {
	Name      string
	arHandle  *AppRes
	kubeRes   *KubeAppRes
	extension *ExtensionParam
	template  AppTemplate
}

func NewDeployWorker(name, namespace, kind string, ar *AppRes, eparam *ExtensionParam, tpl AppTemplate) *DeployWorker {
	return &DeployWorker{
		Name:      name,
		arHandle:  ar,
		kubeRes:   NewKubeAppRes(ar.Client, ar.Cluster, namespace, ar.DomainSuffix, kind),
		extension: eparam,
		template:  tpl,
	}
}

func (wk *DeployWorker) Start(templateName string, param AppParam) error {
	beego.Info("deploying application: ", wk.Name)
	err := wk.checkAppRes(param.Name)
	if err != nil {
		return err
	}
	app, err := wk.arHandle.Appmodel.GetAppByName(wk.arHandle.Cluster, wk.kubeRes.Namespace, param.Name)
	if err == nil {
		return wk.updateAppRes(*app)
	} else {
		if err != orm.ErrNoRows {
			return err
		}
		return wk.createAppRes(templateName, param)
	}
}

//check app res, maybe delete some data
func (wk *DeployWorker) checkAppRes(appname string) error {
	//check app name uniqueness
	exoticapps, err := wk.arHandle.Appmodel.GetExoticAppListByName(wk.arHandle.Cluster, wk.kubeRes.Namespace, appname)
	if err != nil {
		return err
	}
	if len(exoticapps) != 0 {
		var exoticns []string
		if wk.extension.Force {
			//check right
			for _, app := range exoticapps {
				exoticns = append(exoticns, app.Namespace)
			}
			if len(exoticns) == 0 {
				//uninstall
				for _, app := range exoticapps {
					beego.Warn(fmt.Sprintf("deleting application(%s), cluster(%s), namespace(%s), and you have right to do it...", appname, wk.arHandle.Cluster, app.Namespace))
					if err = wk.arHandle.DeleteApp(app.Namespace, app.Name); err != nil {
						return fmt.Errorf("the application(%s) is existed in namespace %v of cluster %v, and delete old application failed: %s",
							appname, app.Namespace, wk.arHandle.Cluster, err.Error())
					}
					beego.Warn(fmt.Sprintf("delete application(%s), cluster(%s), namespace(%s) successfully, and you have right to do it...", appname, wk.arHandle.Cluster, app.Namespace))
				}
			}
		} else {
			//no right
			for _, app := range exoticapps {
				exoticns = append(exoticns, app.Namespace)
			}
		}
		if len(exoticns) != 0 {
			return fmt.Errorf("the application(%s) is existed in namespace %v of cluster %v, and you have no right to cover the old application!", appname, exoticns, wk.arHandle.Cluster)
		}
	}

	return nil
}

func (wk *DeployWorker) updateAppRes(app models.ZcloudApplication) error {
	//delete possible resource
	beego.Info("delete possible deploy and pod resource: ", wk.arHandle.Cluster, wk.kubeRes.Namespace, app.Name, app.Kind)
	wk.deleteApplication(app.Name)
	_, err := wk.arHandle.ReconfigureApp(app, wk.template)
	if err != nil {
		return err
	}
	return nil
}

func (wk *DeployWorker) createAppRes(templateName string, param AppParam) error {
	// create new app resource
	app, err := wk.createKubeAppRes(templateName, param)
	if err != nil {
		return err
	}
	err = wk.arHandle.Appmodel.CreateApp(*app)
	if err != nil {
		wk.kubeRes.DeleteAppResource(wk.template, app.PodVersion)
		wk.arHandle.Appmodel.DeleteApp(*app)
		return err
	}
	if wk.extension != nil {
		if wk.extension.Patcher != nil {
			wk.extension.Patcher(*app)
		}
	}
	return nil
}

func (wk *DeployWorker) createKubeAppRes(templateName string, param AppParam) (*models.ZcloudApplication, error) {
	app, err := wk.template.GenerateAppObject(wk.arHandle.Cluster, wk.kubeRes.Namespace, templateName, wk.arHandle.DomainSuffix)
	if err != nil {
		return nil, err
	}
	//delete possible resource
	beego.Info("delete possible deploy and pod resource: ", wk.arHandle.Cluster, wk.kubeRes.Namespace, param.Name, app.Kind)
	wk.deleteApplication(param.Name)
	beego.Info("create resource: ", wk.arHandle.Cluster, wk.kubeRes.Namespace, param.Name, app.Kind)
	if err := wk.kubeRes.CreateAppResource(wk.template, app.PodVersion); err != nil {
		return nil, err
	}
	return app, nil
}

func (wk *DeployWorker) deleteApplication(appname string) {
	filter := utils.NewFilterQuery(false).SetFilter("name", appname, utils.FilterOperatorEqual)
	defFilter := utils.NewDefaultFilter().AppendFilter("namespace", wk.kubeRes.Namespace, utils.FilterOperatorEqual)
	res, err := wk.arHandle.Appmodel.GetAppList(defFilter, filter)
	if err != nil {
		beego.Error("deleteApplication error: ", err.Error())
		return
	}
	applist := res.List.([]models.ZcloudApplication)
	ar := *wk.arHandle
	for _, app := range applist {
		exist, err := wk.kubeRes.CheckAppIsExisted(app.Name, app.PodVersion)
		if err == nil && exist && wk.arHandle.Cluster != app.Cluster {
			ar.Cluster = app.Cluster
			err = (&ar).DeleteApp(app.Namespace, app.Name)
			if err != nil {
				beego.Info(fmt.Sprintf("delete unsuitable application(%s/%s/%s) failed: %v!", app.Cluster, app.Name, app.PodVersion, err))
			} else {
				beego.Info(fmt.Sprintf("delete unsuitable application(%s/%s/%s) successfully!", app.Cluster, app.Name, app.PodVersion))
			}
		}
	}
}

func getDefaultPullSecret(cluster string) (string, error) {
	cinfo, err := dao.GetCluster(cluster)
	if err != nil {
		return "", err
	}
	harbor, err := dao.GetHarbor(cinfo.Registry)
	if err != nil {
		return "", err
	}
	return "harbor-" + harbor.HarborName, nil
}
