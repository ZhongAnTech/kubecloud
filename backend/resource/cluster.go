package resource

import (
	"fmt"
	"time"

	"github.com/astaxie/beego"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/common/utils"
	"kubecloud/common/validate"
)

type ClusterList struct {
	Clusters []Cluster `json:"clusters"`
}

type ClusterLoadBalancer struct {
	DomainName string `json:"domain_name"`
	IP         string `json:"ip"`
	Port       string `json:"port"`
}

type Cluster struct {
	models.ZcloudCluster
	DomainSuffixs      []*models.ZcloudClusterDomainSuffix `json:"domain_suffixs"`
	Masters            []NodeInfo                          `json:"master"`
	LoadbalancerConfig ClusterLoadBalancer                 `json:"lb_config"`
	CreateAt           string                              `json:"create_at"`
	UpdateAt           string                              `json:"update_at"`
}

type DeployClusterInfo struct {
	Name               string              `json:"cluster_id"`
	Registry           string              `json:"registry"`
	KubeVersion        string              `json:"kube_version"`
	LoadbalancerConfig ClusterLoadBalancer `json:"lb_config"`
	KubeServiceAddress string              `json:"kube_service_addresses"`
	KubePodSubnet      string              `json:"kube_pods_subnet"`
}

func (cluster Cluster) Verify() error {
	if err := validate.ValidateString(cluster.Name); err != nil {
		return fmt.Errorf("cluster name is not right: %v", err.Error())
	}
	if err := validate.ValidateString(cluster.Tenant); err != nil {
		return fmt.Errorf("cluster tenant is not right: %v", err.Error())
	}
	if cluster.Registry == "" {
		return fmt.Errorf("registry uuid must be given!")
	}
	if !dao.HarborIsExist(cluster.Tenant, cluster.Registry) {
		return fmt.Errorf("default registry(%s) is not existed!", cluster.Registry)
	}
	return nil
}

func ClusterIsExistInTenant(tenant string, fieldValue ...string) (bool, error) {
	clusterParamList := []string{"name", "display_name", "env"}
	if len(fieldValue) != len(clusterParamList) {
		return false, fmt.Errorf("cluster param list num err")
	}
	for i, clusterParam := range clusterParamList {
		if dao.IsClusterParamExistInTenant(tenant, clusterParam, fieldValue[i]) {
			return true, nil
		}
	}
	return false, nil
}

func GetClusterSimpleDetail(cluster string) (*models.ZcloudCluster, error) {
	return dao.GetCluster(cluster)
}

func GetClusterDetail(clusterId string) (*Cluster, error) {
	item, err := dao.GetCluster(clusterId)
	if err != nil {
		return nil, err
	}

	lb_config := ClusterLoadBalancer{
		DomainName: item.LoadbalancerDomainName,
		IP:         item.LoadbalancerIP,
		Port:       item.LoadbalancerPort,
	}
	kubeVersion := GetKubeVersion(item.ClusterId, item.KubeVersion)
	if kubeVersion != item.KubeVersion && kubeVersion != "" {
		item.KubeVersion = kubeVersion
		if err := dao.UpdateCluster(*item); err != nil {
			beego.Error("update kube cluster version failed:", err)
		}
	}
	detail := Cluster{
		LoadbalancerConfig: lb_config,
	}
	detail.ZcloudCluster = *item
	domainSuffixList, err := dao.GetClusterDomainSuffixList(item.ClusterId)
	if err != nil {
		return nil, err
	}
	detail.DomainSuffixs = domainSuffixList

	detail.CreateAt = item.CreateAt.Format("2006-01-02 15:04:05")
	detail.UpdateAt = item.UpdateAt.Format("2006-01-02 15:04:05")

	return &detail, nil
}

func GetClusterList(filter *utils.FilterQuery) (*utils.QueryResult, error) {
	clusters := []Cluster{}
	items, err := dao.GetClusterListByFilter(filter)
	if err != nil {
		return nil, err
	}
	for _, item := range items.List.([]models.ZcloudCluster) {
		cluster := Cluster{}
		cluster.ZcloudCluster = item
		domainSuffixList, err := dao.GetClusterDomainSuffixList(item.ClusterId)
		if err != nil {
			return nil, err
		}
		cluster.DomainSuffixs = domainSuffixList
		cluster.Certificate = ""
		cluster.CreateAt = item.CreateAt.Format("2006-01-02 15:04:05")
		cluster.UpdateAt = item.UpdateAt.Format("2006-01-02 15:04:05")
		clusters = append(clusters, cluster)
	}
	items.List = clusters

	return items, nil
}

func CreateCluster(cluster Cluster) (*Cluster, error) {
	exist, err := ClusterIsExistInTenant(cluster.Tenant, cluster.Name, cluster.DisplayName, cluster.Env)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("cluster name/chinese name/env exists in tenant, please check")
	}
	var defaultDomainSuffix string
	for _, domainSuffix := range cluster.DomainSuffixs {
		if domainSuffix.IsDefault {
			defaultDomainSuffix = domainSuffix.DomainSuffix
		}
		if err := dao.AddClusterDomainSuffix(&models.ZcloudClusterDomainSuffix{
			Cluster:      cluster.ClusterId,
			DomainSuffix: domainSuffix.DomainSuffix,
			IsDefault:    domainSuffix.IsDefault,
			Addons:       models.NewAddons(),
		}); err != nil {
			return nil, err
		}
	}

	clusterModel := models.ZcloudCluster{
		Name:                   cluster.Name,
		Tenant:                 cluster.Tenant,
		ClusterId:              cluster.ClusterId,
		DisplayName:            cluster.DisplayName,
		Registry:               cluster.Registry,
		RegistryName:           cluster.RegistryName,
		ImagePullAddr:          cluster.ImagePullAddr,
		PrometheusAddr:         cluster.PrometheusAddr,
		DockerVersion:          cluster.DockerVersion,
		NetworkPlugin:          cluster.NetworkPlugin,
		DomainSuffix:           defaultDomainSuffix,
		Certificate:            cluster.Certificate,
		Status:                 models.ClusterStatusToBeConfigured,
		KubeVersion:            cluster.KubeVersion,
		LoadbalancerDomainName: cluster.LoadbalancerConfig.DomainName,
		LoadbalancerIP:         cluster.LoadbalancerConfig.IP,
		LoadbalancerPort:       cluster.LoadbalancerConfig.Port,
		KubeServiceAddress:     cluster.KubeServiceAddress,
		KubePodSubnet:          cluster.KubePodSubnet,
		IngressSLB:             cluster.IngressSLB,
		Env:                    cluster.Env,
		Usage:                  cluster.Usage,
		LabelPrefix:            cluster.LabelPrefix,
		ConfigRepo:             cluster.ConfigRepo,
		ConfigRepoBranch:       fmt.Sprintf("%s-%s", cluster.Tenant, cluster.Name),
		ConfigRepoToken:        cluster.ConfigRepoToken,
		LastCommitId:           cluster.LastCommitId,
		Addons:                 models.NewAddons(),
	}

	if cluster.Certificate != "" {
		clusterModel.Status = models.ClusterStatusPending
		if err := AddClusterK8sConfig(cluster.ClusterId, cluster.Certificate); err != nil {
			return nil, err
		}
		if _, err := service.UpdateClientset(cluster.ClusterId); err != nil {
			return nil, err
		}
		go cm.StartControllers(cluster.ClusterId)
	}

	if err := dao.CreateCluster(clusterModel); err != nil {
		return nil, err
	}

	return GetClusterDetail(cluster.ClusterId)
}

func UpdateCluster(cluster Cluster) (*Cluster, error) {
	//if err := dao.DeleteClusterDomainSuffix(cluster.ClusterId); err != nil {
	//	return nil, err
	//}
	var defaultDomainSuffix string
	for _, domainSuffix := range cluster.DomainSuffixs {
		if domainSuffix.IsDefault {
			defaultDomainSuffix = domainSuffix.DomainSuffix
		}
		if err := dao.AddClusterDomainSuffix(&models.ZcloudClusterDomainSuffix{
			Cluster:      cluster.ClusterId,
			DomainSuffix: domainSuffix.DomainSuffix,
			IsDefault:    domainSuffix.IsDefault,
			Addons:       models.NewAddons(),
		}); err != nil {
			return nil, err
		}
	}

	item, err := dao.GetClusterByTenant(cluster.Tenant, cluster.Name)
	if err != nil {
		return nil, err
	}

	if cluster.DisplayName != "" {
		item.DisplayName = cluster.DisplayName
	}
	if cluster.Registry != "" {
		item.Registry = cluster.Registry
	}
	if cluster.RegistryName != "" {
		item.RegistryName = cluster.RegistryName
	}
	if cluster.DockerVersion != "" {
		item.DockerVersion = cluster.DockerVersion
	}
	if cluster.NetworkPlugin != "" {
		item.NetworkPlugin = cluster.NetworkPlugin
	}
	if cluster.DomainSuffix != "" {
		item.DomainSuffix = cluster.DomainSuffix
	}
	if cluster.IngressSLB != "" {
		item.IngressSLB = cluster.IngressSLB
	}
	if cluster.Status != "" {
		item.Status = cluster.Status
	}
	if cluster.Env != "" {
		item.Env = cluster.Env
	}
	if cluster.Usage != "" {
		item.Usage = cluster.Usage
	}
	if cluster.KubeVersion != "" && item.KubeVersion != cluster.KubeVersion {
		item.KubeVersion = cluster.KubeVersion
	}
	if cluster.LabelPrefix != "" {
		item.LabelPrefix = cluster.LabelPrefix
	}
	if cluster.ConfigRepo != "" {
		item.ConfigRepo = cluster.ConfigRepo
	}
	if cluster.ConfigRepoToken != "" {
		item.ConfigRepoToken = cluster.ConfigRepoToken
	}
	if cluster.LastCommitId != "" {
		item.LastCommitId = cluster.LastCommitId
	}
	item.ConfigRepoBranch = fmt.Sprintf("%s-%s", cluster.Tenant, cluster.Name)
	if defaultDomainSuffix != "" {
		item.DomainSuffix = defaultDomainSuffix
	}
	item.ImagePullAddr = cluster.ImagePullAddr
	item.PrometheusAddr = cluster.PrometheusAddr
	item.TillerHost = cluster.TillerHost
	if cluster.Certificate != "" {
		item.Status = models.ClusterStatusPending
		item.Certificate = cluster.Certificate
		if err = AddClusterK8sConfig(item.ClusterId, cluster.Certificate); err != nil {
			return nil, err
		}
		if _, err = service.UpdateClientset(item.ClusterId); err != nil {
			return nil, err
		}
		go cm.StartControllers(item.ClusterId)
	}

	item.Addons = item.Addons.UpdateAddons()

	if err := dao.UpdateCluster(*item); err != nil {
		return nil, err
	}

	return GetClusterDetail(cluster.ClusterId)
}

func SetClusterCertificate(clusterId, certificate string) error {
	item, err := dao.GetCluster(clusterId)
	if err != nil {
		return err
	}

	item.Certificate = certificate
	item.Status = models.ClusterStatusUpdating
	if err = AddClusterK8sConfig(item.ClusterId, certificate); err != nil {
		return err
	}

	if _, err = service.UpdateClientset(item.ClusterId); err != nil {
		return err
	}

	item.Addons = item.Addons.UpdateAddons()

	if err = dao.UpdateCluster(*item); err != nil {
		return err
	}

	go cm.StartControllers(item.ClusterId)

	return nil
}

func DeleteCluster(clusterId string) error {
	// (TODO) should check before delete
	if err := dao.DeleteNodesByClusterName(clusterId); err != nil {
		return err
	}

	if err := dao.DeleteClusterDomainSuffix(clusterId); err != nil {
		return err
	}

	return dao.DeleteCluster(clusterId)
}

func GetKubeVersion(cluster, defversion string) string {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error("get version failed:", err)
		return defversion
	}
	ver, err := client.Discovery().ServerVersion()
	if err != nil {
		beego.Error("get version failed:", err)
		return defversion
	}
	return ver.GitVersion
}

func CheckClusterApi() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		items, err := dao.GetClusterList()
		if err != nil {
			beego.Error("CheckClusterApi failed: ", err.Error())
			return
		}
		for _, item := range items {
			if item.Certificate != "" {
				newStatus := models.ClusterStatusError
				for i := 0; i < 4; i++ {
					client, err := service.GetClientset(item.ClusterId)
					if err == nil {
						_, err = client.CoreV1().Namespaces().Get("default", meta_v1.GetOptions{})
					}
					if err != nil {
						beego.Error("CheckClusterApi failed:", item.Name, err.Error())
						time.Sleep(time.Duration(30) * time.Second)
						continue
					}
					newStatus = models.ClusterStatusRunning
					break
				}
				if err = dao.UpdateClusterStatus(item.ClusterId, newStatus); err != nil {
					beego.Error("CheckClusterApi failed when update cluster status:", item.Name, err.Error())
				}
			}
		}
	}
}
