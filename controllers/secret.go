package controllers

import (
	"fmt"

	"kubecloud/backend/resource"
	"kubecloud/common"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
)

const MAX_TLS_FILE_SIZE = 10240

type SecretController struct {
	BaseController
}

type FileSizer interface {
	Size() int64
}

// List get all secret of cluster's namespace
func (sc *SecretController) List() {
	clusterId := sc.GetStringFromPath(":cluster")
	namespace := sc.GetStringFromPath(":namespace")
	filterQuery := sc.GetFilterQuery()

	sHandle, err := resource.NewSecretRes(clusterId, NamespaceListFunc(clusterId, namespace))
	if err != nil {
		beego.Error("Get secrets failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	result, err := sHandle.ListSecret(namespace, filterQuery)
	if err != nil {
		beego.Error("Get secrets failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	sc.ServeResult(NewResult(true, result, ""))
}

// Create create a secret
func (sc *SecretController) Create() {
	clusterId := sc.GetStringFromPath(":cluster")
	namespace := sc.GetStringFromPath(":namespace")

	request := new(resource.SecretRequest)
	sc.DecodeJSONReq(&request)

	sHandle, err := resource.NewSecretRes(clusterId, nil)
	if err != nil {
		beego.Error("Create secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+request.Name+".")
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	err = sHandle.Validate(request)
	if err != nil {
		beego.Error("Create secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+request.Name+".")
		sc.ServeError(err)
		return
	}
	err = sHandle.CreateSecret(namespace, request)
	if err != nil {
		beego.Error("Create secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace+".")
		sc.ServeError(err)
		return
	}

	beego.Info("Create secret successfully:", "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+request.Name+".")
	sc.Data["json"] = NewResult(true, nil, "")
	sc.ServeJSON()
}

// Update update a secret
func (sc *SecretController) Update() {
	clusterId := sc.GetStringFromPath(":cluster")
	namespace := sc.GetStringFromPath(":namespace")
	name := sc.GetStringFromPath(":secret")

	request := new(resource.SecretRequest)
	sc.DecodeJSONReq(request)
	request.Name = name

	sHandle, err := resource.NewSecretRes(clusterId, nil)
	if err != nil {
		beego.Error("Update secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+name+".")
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	err = sHandle.Validate(request)
	if err != nil {
		beego.Error("Update secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+name+".")
		sc.ServeError(err)
		return
	}

	err = sHandle.UpdateSecret(namespace, request)
	if err != nil {
		beego.Error("Update secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+name+".")
		sc.ServeError(err)
		return
	}

	beego.Info("Update secret successfully:", "cluster: "+clusterId+",", "namespace: "+namespace, "secret name: "+name+".")
	sc.Data["json"] = NewResult(true, nil, "")
	sc.ServeJSON()
}

// Delete delete a secret
func (sc *SecretController) Delete() {
	clusterId := sc.GetStringFromPath(":cluster")
	namespace := sc.GetStringFromPath(":namespace")
	name := sc.GetStringFromPath(":secret")
	secret, err := resource.NewSecretRes(clusterId, nil)
	if err != nil {
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secretname: "+name+".")
		return
	}
	ir, err := resource.NewIngressRes(clusterId, nil, NamespaceListFunc(clusterId, namespace))
	filterQuery := utils.NewFilterQuery(false).SetFilter("secret_name", name, utils.FilterOperatorEqual)
	result, err := ir.ListIngresses(clusterId, namespace, "", filterQuery)
	if err != nil {
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secretname: "+name+".")
		return
	}
	if list, ok := result.List.([]resource.Ingress); ok && len(list) != 0 {
		beego.Error("Delete secret failed: "+"the secret has been used by some ingresses, can not be delete!", "cluster: "+clusterId+",", "namespace: "+namespace, "secretname: "+name+".")
		beego.Debug("ingress:", list, "hava used this secret!")
		sc.ServeError(common.NewConflict().SetCause(fmt.Errorf(`the secret has been used by some ingress, can not be delete`)))
		return
	}
	err = secret.DeleteSecret(namespace, name)
	if err != nil {
		sc.ServeError(common.NewInternalServerError().SetCause(err))
		beego.Error("Delete secret failed: "+err.Error(), "cluster: "+clusterId+",", "namespace: "+namespace, "secretname: "+name+".")
		return
	}

	sc.Data["json"] = NewResult(true, nil, "")
	sc.ServeJSON()
	beego.Info("Delete secret successfully!", "cluster: "+clusterId+",", "namespace: "+namespace+",", "secretname: "+name+"!")
}
