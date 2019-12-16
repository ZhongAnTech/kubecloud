package controllers

import (
	"kubecloud/backend/dao"
	"kubecloud/backend/resource"
	"kubecloud/common"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type IngressController struct {
	BaseController
}

//get all ingesses of cluster's namespace
func (ic *IngressController) List() {
	clusterId := ic.GetStringFromPath(":cluster")
	namespace := ic.GetStringFromPath(":namespace")
	name := ic.GetString("appname")
	filterQuery := ic.GetFilterQuery()
	emptyRes := &utils.QueryResult{List: []resource.IngressRule{}}
	if name != "" {
		svc, err := dao.NewK8sServiceModel().Get(clusterId, namespace, name, "")
		if err != nil {
			if err == orm.ErrNoRows {
				beego.Warn(clusterId, namespace, name, "application has no service!")
				ic.Data["json"] = NewResult(true, emptyRes, "")
				ic.ServeJSON()
			} else {
				ic.ServeError(common.NewInternalServerError().SetCause(err))
				beego.Error("Get service ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "appname: "+name)
			}
			return
		}
		name = svc.Name
	}
	ing, err := resource.NewIngressRes(clusterId, nil, NamespaceListFunc(clusterId, namespace))
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Get ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace)
		return
	}
	result, err := ing.ListIngresses(clusterId, namespace, name, filterQuery)
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Get ingresses failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		return
	}
	ic.Data["json"] = NewResult(true, result, "")
	ic.ServeJSON()
}

//create a ingess
func (ic *IngressController) Create() {
	clusterId := ic.GetStringFromPath(":cluster")
	namespace := ic.GetStringFromPath(":namespace")

	ing, err := resource.NewIngressRes(clusterId, nil, nil)
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Create ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace)
		return
	}
	ingress := resource.NewEmptyIngress(namespace)
	ic.DecodeJSONReq(&ingress)
	err = ing.Validate(&ingress)
	if err != nil {
		ic.ServeError(common.NewBadRequest().SetCause(err))
		beego.Error("Create ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		return
	}
	err = ing.CreateIngress(&ingress)
	if err != nil {
		beego.Error("Create ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		ic.ServeError(err)
		return
	}

	beego.Info("Create ingress successfully:", "cluster: "+clusterId+",", "namespace: "+namespace, "host: "+ingress.Host+".")
	ic.Data["json"] = NewResult(true, nil, "")
	ic.ServeJSON()
}

//update a ingess
func (ic *IngressController) Update() {
	clusterId := ic.GetStringFromPath(":cluster")
	namespace := ic.GetStringFromPath(":namespace")
	id, err := ic.GetInt64FromPath(":ingressID")
	if err != nil {
		beego.Error("update ingess failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		ic.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	ing, err := resource.NewIngressRes(clusterId, nil, nil)
	if err != nil {
		beego.Error("Update ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	ingress := resource.NewEmptyIngress(namespace)
	ic.DecodeJSONReq(&ingress)
	err = ing.Validate(&ingress)
	if err != nil {
		beego.Error("Update ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		ic.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	if ingress.Protocol == resource.APP_PROTOCOL_HTTP && ingress.SecretName != "" {
		// set empty
		ingress.SecretName = ""
	}
	if err := ing.UpdateIngress(id, ingress); err != nil {
		beego.Error("Update ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		ic.ServeError(err)
		return
	}
	ic.Data["json"] = NewResult(true, nil, "")
	ic.ServeJSON()
	beego.Info("Update ingress successfully:", "cluster: "+clusterId+",", "namespace: "+namespace, "host: "+ingress.Host+".")
}

//Delete a ingess
func (ic *IngressController) Delete() {
	clusterId := ic.GetStringFromPath(":cluster")
	namespace := ic.GetStringFromPath(":namespace")
	id, err := ic.GetInt64FromPath(":ingressID")
	if err != nil {
		ic.ServeError(common.NewBadRequest().SetCause(err))
		beego.Error("Delete ingess failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		return
	}
	ing, err := resource.NewIngressRes(clusterId, nil, nil)
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "id: ", id, ".")
		return
	}
	err = ing.DeleteIngress(namespace, id)
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "ingressid: ", id, ".")
		return
	}

	ic.Data["json"] = NewResult(true, nil, "")
	ic.ServeJSON()
	beego.Info("Delete  successfully!", "cluster: "+clusterId+",", "namespace: "+namespace+",", "ingressid: ", id, "!")
}

func (ic *IngressController) Inspect() {
	clusterId := ic.GetStringFromPath(":cluster")
	namespace := ic.GetStringFromPath(":namespace")
	id, err := ic.GetInt64FromPath(":ingressID")
	if err != nil {
		ic.ServeError(common.NewBadRequest().SetCause(err))
		beego.Error("Get ingess failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		return
	}
	ing, err := resource.NewIngressRes(clusterId, nil, nil)
	if err != nil {
		ic.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Get ingress failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "ingressid: ", id, ".")
		return
	}
	result, err := ing.GetIngressDetail(namespace, id)
	if err != nil {
		ic.Data["json"] = NewResult(false, result, err.Error())
		ic.ServeJSON()
		beego.Error("Get ingress backend failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "ingressid: ", id, ".")
		return
	}

	ic.Data["json"] = NewResult(true, result, "")
	ic.ServeJSON()
	beego.Info("Get successfully!", "cluster: "+clusterId+",", "namespace: "+namespace+",", "ingressid: ", id, "!")
}
