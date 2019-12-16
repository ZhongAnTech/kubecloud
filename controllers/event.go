package controllers

import (
	"github.com/astaxie/beego"

	"kubecloud/backend/resource"
	"kubecloud/common"
)

type EventsController struct {
	BaseController
}

func (this *EventsController) Get() {
	clusterName := this.Ctx.Input.Query("cluster_name")
	namespace := this.Ctx.Input.Query("namespace")
	sourceHost := this.Ctx.Input.Query("source_host")
	objectKind := this.Ctx.Input.Query("object_kind")
	objectName := this.Ctx.Input.Query("object_name")
	eventLevel := this.Ctx.Input.Query("event_level")
	limitCount, err := this.GetInt64FromQuery("limit_count")
	if err != nil {
		beego.Error("Parse int error of limit_count: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	events, err := resource.GetEvents(clusterName, namespace, sourceHost, objectKind, objectName, eventLevel, limitCount)
	if err != nil {
		beego.Error("Get all events occur err: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, events, "")
	this.ServeJSON()
}
