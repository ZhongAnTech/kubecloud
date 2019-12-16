package service

import (
	"fmt"
	"path"

	"github.com/astaxie/beego"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeRestConfig(cluster string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", path.Join(beego.AppConfig.String("k8s::configPath"), cluster))
}

// Heapster client
type HeapsterClient interface {
	Get(path string) RequestInterface
}

type InClusterHeapsterClient struct {
	client rest.Interface
}

type RequestInterface interface {
	DoRaw() ([]byte, error)
}

func NewHeapsterClient(cluster string) (HeapsterClient, error) {
	client, err := GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("get client error %v", err)
	}

	return InClusterHeapsterClient{client: client.CoreV1().RESTClient()}, nil
}

func (c InClusterHeapsterClient) Get(path string) RequestInterface {
	return c.client.Get().Prefix("proxy").
		Namespace("kube-system").
		Resource("services").
		Name("heapster").
		Suffix("/api/v1" + path)
}
