package gitops

import (
	"fmt"
	"github.com/astaxie/beego"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubecloud/backend/dao"
)

const (
	configRepoDir = "configRepo"
)

var (
	ConfigRepos = make(map[string]*Git)
	mux         sync.Mutex
)

func CloneClusterConfigRepo() error {
	items, err := dao.GetAllClusters()
	if err != nil {
		glog.Errorf("clone cluster config repo failed: %s", err.Error())
		return err
	}
	dir := "configRepo/"
	for _, item := range items {
		if item.ConfigRepo != "" && item.ConfigRepoBranch != "" && item.ConfigRepoToken != "" {
			dirPath := dir + item.ClusterId
			g := NewGit(dirPath, item.ConfigRepo, item.ConfigRepoBranch, item.ConfigRepoToken)
			if err := g.Clone(); err != nil {
				glog.Errorf("clone config repo of cluster %s failed: %s, dir: %s, repo: %s, branch: %s", item.ClusterId, err.Error(), dirPath, item.ConfigRepo, item.ConfigRepoBranch)
			}
			ConfigRepos[item.ClusterId] = g
			glog.Infof("clone config repo of cluster %s success, dir: %s, repo: %s, branch: %s", item.ClusterId, dirPath, item.ConfigRepo, item.ConfigRepoBranch)
		}
	}
	return nil
}

func CommitK8sResource(clusterId string, resList []interface{}) error {
	mux.Lock()
	startTime := time.Now()
	defer func() {
		mux.Unlock()
		beego.Info(fmt.Sprintf("Finished gitops commit in cluster %v (%v)", clusterId, time.Now().Sub(startTime)))
	}()
	g, ok := ConfigRepos[clusterId]
	if !ok {
		return fmt.Errorf("cluster %v not have config repo in git", clusterId)
	}
	// git pull
	if err := g.Pull(); err != nil {
		return err
	}

	var files []string
	dir := filepath.Join(configRepoDir, clusterId)
	for _, res := range resList {
		var isDeployment bool
		var subDir string
		var fileName string
		switch t := res.(type) {
		case *corev1.Namespace:
			subDir = "namespaces"
			fileName = fmt.Sprintf("%s.yaml", t.Name)
			res.(*corev1.Namespace).TypeMeta = metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			}
			res.(*corev1.Namespace).ObjectMeta.ResourceVersion = ""
		case *corev1.ResourceQuota:
			subDir = "resourcequotas"
			fileName = fmt.Sprintf("%s.yaml", t.Name)
			res.(*corev1.ResourceQuota).TypeMeta = metav1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			}
			res.(*corev1.ResourceQuota).ObjectMeta.ResourceVersion = ""
		case *corev1.ConfigMap:
			subDir = "configmaps"
			fileName = fmt.Sprintf("%s.yaml", t.Name)
			res.(*corev1.ConfigMap).TypeMeta = metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			}
			res.(*corev1.ConfigMap).ObjectMeta.ResourceVersion = ""
		case *appsv1beta1.Deployment:
			isDeployment = true
			ownerName, ok := t.Annotations["owner_name"]
			if !ok {
				err := fmt.Errorf("not owner_name annotation")
				return err
			}
			subDir = fmt.Sprintf("apps/%s/%s", ownerName, t.Namespace)
			fileName = fmt.Sprintf("%s-dept.yaml", t.Name)
			res.(*appsv1beta1.Deployment).TypeMeta = metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1beta1",
			}
			res.(*appsv1beta1.Deployment).ObjectMeta.ResourceVersion = ""
		case *corev1.Service:
			ownerName, ok := t.Annotations["owner_name"]
			if !ok {
				err := fmt.Errorf("not owner_name annotation")
				return err
			}
			version, ok := t.ObjectMeta.Labels["version"]
			if !ok {
				err := fmt.Errorf("not version label")
				return err
			}
			subDir = fmt.Sprintf("apps/%s/%s", ownerName, t.Namespace)
			fileName = fmt.Sprintf("%s-%s-svc.yaml", t.Name, version)
			res.(*corev1.Service).TypeMeta = metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			}
			res.(*corev1.Service).ObjectMeta.ResourceVersion = ""
		case *extensionsv1beta1.Ingress:
			ownerName, ok := t.Annotations["owner_name"]
			if !ok {
				err := fmt.Errorf("not owner_name annotation")
				return err
			}
			version, ok := t.ObjectMeta.Labels["version"]
			if !ok {
				err := fmt.Errorf("not version label")
				return err
			}
			subDir = fmt.Sprintf("apps/%s/%s", ownerName, t.Namespace)
			fileName = fmt.Sprintf("%s-%s-ing.yaml", t.Name, version)
			res.(*extensionsv1beta1.Ingress).TypeMeta = metav1.TypeMeta{
				Kind:       "Ingress",
				APIVersion: "extensions/v1beta1",
			}
			res.(*extensionsv1beta1.Ingress).ObjectMeta.ResourceVersion = ""
		default:
			err := fmt.Errorf("unsupported k8s resource type: %v", res)
			glog.Error(err.Error())
			return err
		}
		yamlData, err := yaml.Marshal(res)
		if err != nil {
			glog.Errorf("marshal yaml error: %s", err.Error())
			return err
		}
		if err := writeYamlFile(yamlData, dir, subDir, fileName, isDeployment); err != nil {
			return err
		}
		files = append(files, filepath.Join(subDir, fileName))
	}
	err := PushCommits(g, files)
	return err
}

func PushCommits(g *Git, files []string) error {
	beego.Info("commit files: ", files)
	msg := fmt.Sprintf("Update %v files", files)
	if err := g.Commit(files, msg); err != nil {
		return err
	}
	// git pull
	if err := g.Pull(); err != nil {
		return err
	}
	// git push
	if err := g.Push(); err != nil {
		return err
	}
	return nil
}

func writeYamlFile(data []byte, dir, subDir, fileName string, isDeployment bool) error {
	exist, err := pathExists(filepath.Join(dir, subDir))
	if err != nil {
		glog.Errorf("get dir error: %s", err.Error())
		return err
	}
	if !exist {
		if err := os.MkdirAll(filepath.Join(dir, subDir), os.ModePerm); err != nil {
			glog.Errorf("mkdir error: %s", err.Error())
			return err
		}
	}
	yamlFile, err := os.OpenFile(filepath.Join(dir, subDir, fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		glog.Errorf("Failed to open the file", err.Error())
		return err
	}
	defer yamlFile.Close()

	if isDeployment {
		recreate := strings.Contains(string(data), `type: Recreate`)
		if recreate {
			data = []byte(strings.ReplaceAll(string(data), `type: Recreate`, "type: Recreate\n    rollingUpdate: null"))
		}
	}

	if _, err := yamlFile.Write(data); err != nil {
		glog.Errorf("Failed to write the file", err.Error())
		return err
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
