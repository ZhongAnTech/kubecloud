package controllers

import (
	"fmt"
	"kubecloud/backend/dao"
	"kubecloud/backend/resource"
	"kubecloud/common"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type AppController struct {
	BaseController
}

type AppDeployModel struct {
	Cluster      string              `json:"cluster"`
	Namespace    string              `json:"namespace"`
	TemplateName string              `json:"template_name"`
	TimeOut      int64               `json:"time_out"`
	AppParams    []resource.AppParam `json:"app_params"`
}

func (this *AppController) List() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	filterQuery := this.GetFilterQuery()
	defaultQuery := &utils.DefaultFilter{}
	listIsEmpty := false
	// filter cluster
	if clusterId == common.AllCluster {
		clusterList, err := dao.GetClusterList()
		if err != nil {
			this.ServeError(common.NewInternalServerError().SetCause(err))
		}
		clusters := []string{}
		for _, cluster := range clusterList {
			clusters = append(clusters, cluster.ClusterId)
		}
		if len(clusters) == 0 {
			listIsEmpty = true
		}
		defaultQuery.AppendFilter("cluster", clusters, utils.FilterOperatorIn)
		clusterId = ""
	} else {
		defaultQuery.AppendFilter("cluster", clusterId, utils.FilterOperatorEqual)
	}

	result := utils.InitQueryResult([]resource.AppItem{}, filterQuery)
	if !listIsEmpty {
		ar, err := resource.NewAppRes(clusterId, NamespaceListFunc(clusterId, namespace))
		if err != nil {
			this.ServeError(common.NewInternalServerError().SetCause(err))
			return
		}
		result, err = ar.GetAppList(namespace, defaultQuery, filterQuery)
		if err != nil {
			beego.Error("get app list failed:", err)
			this.ServeError(common.NewInternalServerError().SetCause(err))
			return
		}
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *AppController) Inspect() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	appname := utils.GenerateStandardAppName(this.Ctx.Input.Param(":app"))
	ar, err := resource.NewAppRes(clusterId, NamespaceListFunc(clusterId, namespace))
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := ar.GetAppDetail(namespace, appname)
	if err != nil {
		if err == orm.ErrNoRows {
			this.ServeError(common.NewNotFound().SetCause(fmt.Errorf("application %s is not existed", appname)))
		} else {
			beego.Error("Get application information failed: " + err.Error())
			this.ServeError(common.NewInternalServerError().SetCause(err))
		}
		return
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *AppController) Log() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	appname := this.GetStringFromPath(":app")

	podName := this.GetString("podName")
	containerName := this.GetString("containerName")

	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	result, err := ar.GetAppPodLog(namespace, appname, podName, containerName)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *AppController) Create() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	force, err := this.GetBool("force", false)
	if err != nil {
		beego.Error("Created application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	prjid, err := this.GetInt64("projectId", resource.DEFAULT_PROJECT_ID)
	if err != nil {
		beego.Error("Created application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	tpl := resource.NewTemplate()
	this.DecodeJSONReq(&tpl)
	if err := tpl.Validate(); err != nil {
		beego.Error("Created application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error("Created application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	eparam := resource.ExtensionParam{
		Force: force,
	}
	err = ar.InstallApp(prjid, namespace, "", tpl, &eparam)
	if err != nil {
		beego.Error("Created application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(err)
		return
	}

	beego.Info("Created and install application successfully,", "cluster: "+clusterId+",", "namespace: "+namespace, "!")
	this.ServeResult(NewResult(true, "", ""))
}

func (this *AppController) Delete() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	appname := this.Ctx.Input.Param(":app")
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		return
	}
	err = ar.DeleteApp(namespace, appname)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		return
	}
	beego.Info("Delete application successfully!", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *AppController) Restart() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	appname := this.Ctx.Input.Param(":app")
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Restart application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		return
	}
	err = ar.Restart(namespace, appname)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Restart application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		return
	}

	beego.Info("Restart application successfully!", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *AppController) Reconfigure() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	appname := this.GetStringFromPath(":app")

	bodyContext := this.Ctx.Input.CopyBody(1 << 32)
	if len(bodyContext) == 0 {
		beego.Info("There is no information to be updated", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(common.NewBadRequest().SetCause(fmt.Errorf("there is no information to be updated")))
		return
	}
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error("Update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	app, err := ar.Appmodel.GetAppByName(clusterId, namespace, appname)
	if err != nil {
		beego.Error("Update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		if err == orm.ErrNoRows {
			this.ServeError(common.NewNotFound().SetCause(err))
		} else if err == orm.ErrMultiRows {
			this.ServeError(common.NewConflict().SetCause(err))
		} else {
			this.ServeError(common.NewInternalServerError().SetCause(err))
		}
		return
	}
	var template resource.AppTemplate
	var deployConf *resource.DeployConfig
	template, err = resource.CreateNativeAppTemplate(*app, string(bodyContext), deployConf)
	if err != nil {
		beego.Error("Update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	appinfo, err := ar.ReconfigureApp(*app, template)
	if err != nil {
		beego.Error("Update application failed for:"+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(err)
		return
	}

	beego.Info("Update application successfully,", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
	this.ServeResult(NewResult(true, appinfo, ""))
}

func (this *AppController) Scale() {
	clusterId := this.GetStringFromPath(":cluster")
	appname := this.GetStringFromPath(":app")
	namespace := this.GetStringFromPath(":namespace")

	scale, err := this.GetInt("scaleBy")
	if err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if !(scale >= common.ReplicasMin && scale <= common.ReplicasMax) {
		err = fmt.Errorf(
			"replicas error: replicas must be an integer and in the range of %v to %v",
			common.ReplicasMin, common.ReplicasMax)
		beego.Error(err)
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error("scale application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := ar.ScaleApp(namespace, appname, scale); err != nil {
		beego.Error("scale application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	beego.Info("scale application succefully,", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *AppController) RollingUpdate() {
	clusterId := this.GetStringFromPath(":cluster")
	appname := this.Ctx.Input.Param(":app")
	namespace := this.Ctx.Input.Param(":namespace")
	var param []resource.ContainerParam

	this.DecodeJSONReq(&param)
	if err := resource.CheckImageValidate(param); err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		beego.Error("rolling update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "!")
		this.ServeError(err)
		return
	}
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error("rolling update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(err)
		return
	}
	if err := ar.RollingUpdateApp(namespace, appname, param); err != nil {
		beego.Error("rolling update application failed for: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
		this.ServeError(err)
		return
	}
	beego.Info("rolling update application succefully", "cluster: "+clusterId+",", "namespace: "+namespace+",", "name: "+appname, "!")
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *AppController) BatchRollingUpdate() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	var apps []resource.RollingUpdateApp
	this.DecodeJSONReq(&apps)

	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error("BatchRollingUpdate failed:", err.Error(), "cluster:", clusterId, "namespace:"+namespace, "apps:", apps)
		this.ServeError(err)
		return
	}
	err = ar.BatchRollingUpdateApp(namespace, apps)
	if err != nil {
		beego.Error("BatchRollingUpdate failed:", err.Error(), "cluster:", clusterId, "namespace:", namespace, "apps:", apps)
		this.ServeError(err)
		return
	}
	beego.Info("batch rolling update application succefully", "cluster: "+clusterId+",", "namespace: "+namespace, "!")
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *AppController) PodInspect() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	appname := this.Ctx.Input.Param(":app")
	podname := this.Ctx.Input.Param(":podname")

	ar, err := resource.NewAppRes(clusterId, NamespaceListFunc(clusterId, namespace))
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := ar.GetAppPodStatus(namespace, appname, podname)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Get application information failed: " + err.Error())
		return
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *AppController) Event() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	appname := this.GetStringFromPath(":app")

	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	result, err := ar.GetAppEvent(namespace, appname)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *AppController) SetAppLabels() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	appname := this.GetStringFromPath(":app")
	appLabels := make(map[string]string)

	this.DecodeJSONReq(&appLabels)
	ar, err := resource.NewAppRes(clusterId, nil)
	if err != nil {
		beego.Error(fmt.Sprintf("set labels for application(%s/%s/%s) failed: %v", clusterId, namespace, appname, err))
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := ar.SetLabels(namespace, appname, appLabels); err != nil {
		beego.Error(fmt.Sprintf("set labels for application(%s/%s/%s) failed: %v", clusterId, namespace, appname, err))
		switch err.(type) {
		case *common.Error:
			this.ServeError(err)
		default:
			this.ServeError(common.NewInternalServerError().SetCause(err))
		}
		return
	}
	beego.Info(fmt.Sprintf("set labels for application(%s/%s/%s) successfully!", clusterId, namespace, appname))
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}
