package controllers

import (
	"fmt"

	"github.com/astaxie/beego"

	"kubecloud/backend/resource"
	"kubecloud/common"
)

type ServiceController struct {
	BaseController
}

func (this *ServiceController) List() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	if clusterId == "" || namespace == "" {
		beego.Error("cluster: " + clusterId + " or namespace: " + namespace + " is not right, can not be empty")
		err := fmt.Errorf("invalid cluster or namespace")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	srHandle := resource.NewServiceRes(clusterId, NamespaceListFunc(clusterId, namespace))
	result, err := srHandle.GetServiceList(namespace)
	if err != nil {
		this.Data["json"] = NewResult(false, result, err.Error())
	} else {
		this.Data["json"] = NewResult(true, result, "")
	}
	this.ServeJSON()
}

func (this *ServiceController) Inspect() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	name := this.Ctx.Input.Param(":service")
	if clusterId == "" || namespace == "" {
		beego.Error("cluster: " + clusterId + " or namespace: " + namespace + " is not right, can not be empty")
		err := fmt.Errorf("cluster or namespace can not be empty")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	if name == "" {
		beego.Error("service name can not be empty")
		err := fmt.Errorf("service name can not be empty")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	srHandle := resource.NewServiceRes(clusterId, nil)
	result, err := srHandle.GetService(namespace, name, "")
	if err != nil {
		beego.Error("get service"+"("+clusterId, namespace, name+")"+"information failed for", err)
		this.Data["json"] = NewResult(false, nil, err.Error())
	} else {
		this.Data["json"] = NewResult(true, result, "")
	}

	this.ServeJSON()
}
