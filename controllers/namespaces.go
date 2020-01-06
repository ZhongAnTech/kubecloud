package controllers

import (
	corev1 "k8s.io/api/core/v1"

	"kubecloud/backend/resource"
	"kubecloud/common"
)

type NamespaceController struct {
	BaseController
}

func (nc *NamespaceController) ListNamespace() {
	cluster := nc.GetStringFromPath(":cluster")

	namesapceRes, err := resource.NewNamespace(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	res, err := namesapceRes.ListNamespace()
	if err != nil {
		nc.ServeError(err)
		return
	}
	nc.ServeResult(NewResult(true, res, ""))
}

func (nc *NamespaceController) CreateNamespace() {
	cluster := nc.GetStringFromPath(":cluster")
	var data corev1.Namespace
	nc.DecodeJSONReq(&data)

	namesapceRes, err := resource.NewNamespace(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	row, err := namesapceRes.CreateNamespace(&data)
	if err != nil {
		nc.ServeError(err)
		return
	}
	nc.ServeResult(NewResult(true, row, ""))
}

func (nc *NamespaceController) NamespaceLabels() {
	cluster := nc.GetStringFromPath(":cluster")
	namespace := nc.GetStringFromPath(":namespace")
	var labels map[string]string
	nc.DecodeJSONReq(&labels)

	namesapceRes, err := resource.NewNamespace(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	res, err := namesapceRes.NamespaceLabels(namespace, labels)
	if err != nil {
		nc.ServeError(err)
		return
	}
	nc.ServeResult(NewResult(true, res, ""))
}

func (nc *NamespaceController) DeleteNamespace() {
	cluster := nc.GetStringFromPath(":cluster")
	namespace := nc.GetStringFromPath(":namespace")

	namesapceRes, err := resource.NewNamespace(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := namesapceRes.NamespaceDelete(namespace); err != nil {
		nc.ServeError(err)
		return
	}
	nc.ServeResult(NewResult(true, nil, ""))
}

func (nc *NamespaceController) GetNamespace() {
	cluster := nc.GetStringFromPath(":cluster")
	namespace := nc.GetStringFromPath(":namespace")

	namesapceRes, err := resource.NewNamespace(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	res, err := namesapceRes.GetNamespace(namespace)
	if err != nil {
		nc.ServeError(err)
		return
	}
	nc.ServeResult(NewResult(true, res, ""))
}
