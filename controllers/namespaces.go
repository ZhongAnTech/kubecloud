package controllers

import (
	"kubecloud/backend/resource"
	"net/http"
)

type NamespaceController struct {
	BaseController
}

func (this *NamespaceController) NamespaceList() {
	clusterId := this.GetStringFromPath(":cluster")
	filterQuery := this.GetFilterQuery()
	res, err := resource.NamespaceGetAll(clusterId, filterQuery)
	if err != nil {
		this.ServeError(err)
		return
	}
	this.ServeResult(NewResult(true, res, ""))
}

func (this *NamespaceController) Create() {
	clusterId := this.GetStringFromPath(":cluster")
	var data resource.NamespaceData
	this.DecodeJSONReq(&data)
	if err := resource.NamespaceValidate(&data); err != nil {
		this.ServeError(err)
		return
	}
	row, err := resource.NamespaceCreate(clusterId, &data)
	if err != nil {
		this.ServeError(err)
		return
	}
	this.SetStatus(http.StatusCreated)
	this.ServeResult(NewResult(true, row, ""))
}

func (this *NamespaceController) Update() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	var data resource.NamespaceData
	this.DecodeJSONReq(&data)
	data.Name = namespace
	if err := resource.NamespaceValidate(&data); err != nil {
		this.ServeError(err)
		return
	}
	row, err := resource.NamespaceUpdate(clusterId, &data)
	if err != nil {
		this.ServeError(err)
		return
	}
	this.ServeResult(NewResult(true, row, ""))
}

func (this *NamespaceController) Delete() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	if err := resource.NamespaceDelete(clusterId, namespace); err != nil {
		this.ServeError(err)
		return
	}
	this.ServeResult(NewResult(true, nil, ""))
}

func (this *NamespaceController) Inspect() {
	clusterId := this.GetStringFromPath(":cluster")
	namespace := this.GetStringFromPath(":namespace")
	row, err := resource.NamespaceGetOne(clusterId, namespace)
	if err != nil {
		this.ServeError(err)
		return
	}
	this.ServeResult(NewResult(true, row, ""))
}
