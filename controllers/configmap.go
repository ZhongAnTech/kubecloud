package controllers

import (
	corev1 "k8s.io/api/core/v1"

	"kubecloud/backend/resource"
)

type ConfigMapController struct {
	BaseController
}

func (cc *ConfigMapController) List() {
	clusterId := cc.GetStringFromPath(":cluster")
	namespace := cc.GetStringFromPath(":namespace")
	result, err := resource.ConfigMapList(clusterId, namespace)
	if err != nil {
		cc.ServeError(err)
		return
	}

	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ConfigMapController) Create() {
	clusterId := cc.GetStringFromPath(":cluster")
	namespace := cc.GetStringFromPath(":namespace")

	var configMapSpec corev1.ConfigMap
	cc.DecodeJSONReq(&configMapSpec)
	configMapSpec.ObjectMeta.Namespace = namespace
	if configMapSpec.ObjectMeta.Annotations == nil {
		configMapSpec.ObjectMeta.Annotations = make(map[string]string)
	}
	result, err := resource.ConfigMapCreate(clusterId, &configMapSpec)
	if err != nil {
		cc.ServeError(err)
		return
	}

	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ConfigMapController) Inspect() {
	clusterId := cc.GetStringFromPath(":cluster")
	namespace := cc.GetStringFromPath(":namespace")
	name := cc.GetStringFromPath(":configmap")

	result, err := resource.ConfigMapInspect(clusterId, namespace, name)
	if err != nil {
		cc.ServeError(err)
		return
	}
	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ConfigMapController) Update() {
	clusterId := cc.GetStringFromPath(":cluster")
	namespace := cc.GetStringFromPath(":namespace")
	name := cc.GetStringFromPath(":configmap")

	var configData corev1.ConfigMap
	cc.DecodeJSONReq(&configData)
	result, err := resource.ConfigMapUpdate(clusterId, namespace, name, &configData)
	if err != nil {
		cc.ServeError(err)
		return
	}
	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ConfigMapController) Delete() {
	clusterId := cc.GetStringFromPath(":cluster")
	namespace := cc.GetStringFromPath(":namespace")
	name := cc.GetStringFromPath(":configmap")

	if err := resource.ConfigMapDelete(clusterId, namespace, name); err != nil {
		cc.ServeError(err)
		return
	}
	cc.Data["json"] = NewResult(true, nil, "")
	cc.ServeJSON()
}
