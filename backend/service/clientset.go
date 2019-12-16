package service

import (
	"github.com/astaxie/beego"
	"path"
	"sync"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	knativetest "knative.dev/pkg/test"
)

var (
	clusterClientsetMapMutex sync.RWMutex
	clusterClientsetMap      = make(map[string]kubernetes.Interface)
)

func findClientset(cluster string) (client kubernetes.Interface, ok bool) {
	clusterClientsetMapMutex.RLock()
	defer clusterClientsetMapMutex.RUnlock()
	client, ok = clusterClientsetMap[cluster]
	return client, ok
}

func newClientset(cluster string) (client kubernetes.Interface, err error) {
	var ok bool
	clusterClientsetMapMutex.Lock()
	defer clusterClientsetMapMutex.Unlock()
	client, ok = clusterClientsetMap[cluster]
	if !ok {
		client, err = clientsetProvider(cluster)
		if err == nil {
			clusterClientsetMap[cluster] = client
		}
	}
	return client, err
}

func GetClientset(cluster string) (client kubernetes.Interface, err error) {
	var ok bool
	client, ok = findClientset(cluster)
	if !ok {
		client, err = newClientset(cluster)
	}
	return client, err
}

func UpdateClientset(cluster string) (client kubernetes.Interface, err error) {
	clusterClientsetMapMutex.Lock()
	defer clusterClientsetMapMutex.Unlock()
	client, err = clientsetProvider(cluster)
	if err != nil {
		return
	}
	clusterClientsetMap[cluster] = client
	return
}

// Tekton

var (
	tektonClientsetMapMutex     sync.RWMutex
	tektonClientsMapForClusters = make(map[string]*versioned.Clientset)
)

func tektonClientProvider(cluster string) (*versioned.Clientset, error) {
	configPath := path.Join(beego.AppConfig.String("k8s::configPath"), cluster)
	cfg, err := knativetest.BuildClientConfig(configPath, "")
	if err != nil {
		beego.Error("build client config for tekton failed:", configPath, cluster, err)
		return nil, err
	}

	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		beego.Error("new client set for tekton failed:", cluster, err)
		return nil, err
	}
	return cs, nil
}

func findTektonClientset(cluster string) (client *versioned.Clientset, ok bool) {
	tektonClientsetMapMutex.RLock()
	defer tektonClientsetMapMutex.RUnlock()
	client, ok = tektonClientsMapForClusters[cluster]
	return client, ok
}

func newTektonClientset(cluster string) (client *versioned.Clientset, err error) {
	tektonClientsetMapMutex.Lock()
	defer tektonClientsetMapMutex.Unlock()
	client, ok := tektonClientsMapForClusters[cluster]
	if !ok {
		client, err = tektonClientProvider(cluster)
		if err == nil {
			tektonClientsMapForClusters[cluster] = client
		}
	}
	return client, err
}

func GetTektonClientset(cluster string) (client *versioned.Clientset, err error) {
	var ok bool
	client, ok = findTektonClientset(cluster)
	if !ok {
		if client, err = newTektonClientset(cluster); err != nil {
			return nil, err
		}
	}
	return client, err
}
