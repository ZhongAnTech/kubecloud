package service

import (
	"fmt"
	"path"
	"sync"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
)

var clientsetProvider func(cluster string) (kubernetes.Interface, error)

var appConfigProvider func() config.Configer

func Init() {
	clientsetProvider = func(cluster string) (kubernetes.Interface, error) {
		configPath := path.Join(beego.AppConfig.String("k8s::configPath"), cluster)
		config, err := clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}

	appConfigProvider = func() config.Configer {
		return beego.AppConfig
	}
}

func InitMock() {
	clientsetProvider = func(cluster string) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	}

	appConfigContext := struct {
		once     sync.Once
		configer config.Configer
	}{}

	appConfigProvider = func() config.Configer {
		appConfigContext.once.Do(func() {
			confDir := "../../conf"
			configPath := ""
			searchPath := []string{"app.local.conf", "app.test.conf"}
			for _, fileName := range searchPath {
				path := path.Join(confDir, fileName)
				if utils.FileExists(path) {
					configPath = path
					break
				}
			}
			if configPath == "" {
				panic(fmt.Sprintf(`config file not found, search path: %v`, searchPath))
			}
			configer, err := config.NewConfig("ini", configPath)
			if err != nil {
				panic(err)
			}
			appConfigContext.configer = configer
		})
		return appConfigContext.configer
	}
}
