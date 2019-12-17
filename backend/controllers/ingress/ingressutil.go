package ingress

import (
	"kubecloud/backend/models"
	commutil "kubecloud/common/utils"

	extensions "k8s.io/api/extensions/v1beta1"
)

func genIngressRecord(cluster string, ing extensions.Ingress) models.K8sIngress {
	record := models.K8sIngress{}
	record.Name = ing.Name
	record.Namespace = ing.Namespace
	record.Cluster = cluster
	record.Annotation = commutil.SimpleJsonMarshal(ing.Annotations, "")
	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		ruleList := genIngressRuleRecords(cluster, ing, rule)
		record.Rules = append(record.Rules, ruleList...)
	}
	return record
}

func genIngressRuleRecords(cluster string, ing extensions.Ingress, rule extensions.IngressRule) []*models.K8sIngressRule {
	recordList := []*models.K8sIngressRule{}
	record := models.K8sIngressRule{}
	record.Namespace = ing.Namespace
	record.Cluster = cluster
	record.IngressName = ing.Name
	record.IsTls = false
	record.Host = rule.Host
	for _, tls := range ing.Spec.TLS {
		for _, h := range tls.Hosts {
			if h == rule.Host {
				record.IsTls = true
				record.SecretName = tls.SecretName
				break
			}
		}
		if record.IsTls {
			break
		}
	}
	// just one path
	if rule.HTTP != nil {
		for _, path := range rule.HTTP.Paths {
			item := models.K8sIngressRule{}
			item = record
			item.Path = path.Path
			item.ServiceName = path.Backend.ServiceName
			item.ServicePort = path.Backend.ServicePort.IntValue()
			recordList = append(recordList, &item)
		}
	}
	return recordList
}
