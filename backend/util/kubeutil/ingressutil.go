package kubeutil

import (
	"fmt"

	"kubecloud/backend/dao"
	"kubecloud/common/utils"

	extensions "k8s.io/api/extensions/v1beta1"
	"kubecloud/backend/util/labels"
)

const (
	DefaultIngressAnnotationKey = "created_default"
)

func DeleteHostFromTLS(TLS []extensions.IngressTLS, delHost string) []extensions.IngressTLS {
	// delete host from tls
	// if tls.hosts is empty, delete tls from Spec.TLS
	newTLS := []extensions.IngressTLS{}
	for _, tls := range TLS {
		exHosts := []string{}
		for _, host := range tls.Hosts {
			if host != delHost {
				exHosts = append(exHosts, host)
			}
		}
		if len(exHosts) != 0 {
			tls.Hosts = exHosts
			newTLS = append(newTLS, tls)
		}
	}
	return newTLS
}

func MergeIngressRuleValue(first, second *extensions.IngressRule) extensions.IngressRule {
	if first.HTTP == nil {
		return *second
	}
	if second.HTTP == nil {
		return *first
	}
	rule := first.DeepCopy()
	for _, spath := range second.HTTP.Paths {
		found := false
		for _, fpath := range first.HTTP.Paths {
			if utils.PathsIsEqual(fpath.Path, spath.Path) {
				found = true
				break
			}
		}
		if !found {
			rule.HTTP.Paths = append(rule.HTTP.Paths, spath)
		}
	}
	return *rule
}

func MergeIngressRuleList(firstList, secondList []extensions.IngressRule) []extensions.IngressRule {
	for _, orule := range secondList {
		index := -1
		for i, rule := range firstList {
			if orule.Host == rule.Host {
				index = i
				break
			}
		}
		if index < 0 {
			firstList = append(firstList, orule)
		} else {
			firstList[index] = MergeIngressRuleValue(&firstList[index], &orule)
		}
	}
	return firstList
}

func MergeIngressTLSList(firstList, secondList []extensions.IngressTLS) []extensions.IngressTLS {
	// check TLS, if not existed in new ing, then add it
	for _, otls := range secondList {
		index := -1
		for i, ntls := range firstList {
			if ntls.SecretName == otls.SecretName {
				index = i
				break
			}
		}
		if index < 0 {
			firstList = deleteHostsFromTLS(firstList, otls)
			firstList = append(firstList, otls)
		} else {
			firstList[index] = MergeIngressTLSHost(&firstList[index], &otls)
		}
	}
	return firstList
}

func MergeIngressTLSHost(first, second *extensions.IngressTLS) extensions.IngressTLS {
	tls := first.DeepCopy()
	for _, shost := range second.Hosts {
		found := true
		for _, nhost := range first.Hosts {
			if shost == nhost {
				break
			}
		}
		if !found {
			tls.Hosts = append(tls.Hosts, shost)
		}
	}
	return *tls
}

func CheckIngressRule(cluster, namespace string, rules []extensions.IngressRule) error {
	ruleModel := dao.NewIngressRuleModel()
	for _, rule := range rules {
		if err := ruleModel.CheckHostUnique(cluster, namespace, rule.Host); err != nil {
			return err
		}
		if rule.HTTP == nil {
			continue
		}
		paths := []string{}
		for i, path := range rule.HTTP.Paths {
			paths = append(paths, path.Path)
			for j, item := range rule.HTTP.Paths {
				if i != j && utils.PathsIsEqual(path.Path, item.Path) {
					return fmt.Errorf("the path in the ingress rule is duplicated!")
				}
			}
		}
	}
	return nil
}

func SetCreatedDefaultAnno(ing *extensions.Ingress) *extensions.Ingress {
	if ing == nil {
		return nil
	}
	// set default annotation
	labels.AddLabel(ing.ObjectMeta.Annotations, DefaultIngressAnnotationKey, "true")
	return ing
}

func DeleteCreatedDefaultAnno(ing *extensions.Ingress) *extensions.Ingress {
	if ing == nil {
		return nil
	}
	if ing.ObjectMeta.Annotations == nil {
		return ing
	}
	// set default annotation
	delete(ing.ObjectMeta.Annotations, DefaultIngressAnnotationKey)
	return ing
}

func IngressIsCreatedDefault(ing *extensions.Ingress) bool {
	if ing == nil {
		return false
	}
	if ing.ObjectMeta.Annotations == nil {
		return false
	}
	return (ing.ObjectMeta.Annotations[DefaultIngressAnnotationKey] == "true")
}

func deleteHostsFromTLS(TLSList []extensions.IngressTLS, one extensions.IngressTLS) []extensions.IngressTLS {
	for _, host := range one.Hosts {
		TLSList = DeleteHostFromTLS(TLSList, host)
	}
	return TLSList
}
