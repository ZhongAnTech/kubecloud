package util

import (
	"kubecloud/backend/dao"
)

var filterNamespaces = []string{"default", "kube-system", "kube-public", "tekton-pipelines"}

func GetAnnotationStringValue(key string, ann map[string]string, def string) string {
	if ann == nil {
		return def
	}
	if value, exist := ann[key]; exist {
		return value
	}
	return def
}

func GetAnnotationBoolValue(key string, ann map[string]string, def bool) bool {
	if ann == nil {
		return def
	}
	if value, exist := ann[key]; exist {
		return value == "true" || value == "TRUE"
	}
	return def
}

func FilterNamespace(cluster string, namespace string) bool {
	for _, ns := range filterNamespaces {
		if ns == namespace {
			return true
		}
	}
	return !dao.NamespaceExists(cluster, namespace)
}
