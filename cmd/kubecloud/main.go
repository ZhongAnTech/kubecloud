package main

import (
	"flag"
	"runtime"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	_ "github.com/astaxie/beego/session/mysql"

	kubelogs "k8s.io/apiserver/pkg/util/logs"
	"kubecloud/backend/controllermanager"
	_ "kubecloud/backend/controllermanager/register"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"
	"kubecloud/backend/service"
	"kubecloud/gitops"
	"kubecloud/routers"
)

func init() {
	service.Init()

	// for glog
	kubelogs.InitLogs()
	defer kubelogs.FlushLogs()
	flag.Parse()

	logFilename := beego.AppConfig.String("log::logfile")
	logLevel := beego.AppConfig.String("log::level")
	logSeparate := beego.AppConfig.String("log::separate")
	if logFilename == "" {
		logFilename = "log/kubecloud.log"
	}
	if logLevel == "" {
		logLevel = "7"
	}
	if logSeparate == "" {
		logSeparate = "[\"error\"]"
	}
	logconfig := `{
		"filename": "` + logFilename + `",
		"level": ` + logLevel + `,
		"separate": ` + logSeparate + `
	}`
	logs.SetLogger(logs.AdapterMultiFile, logconfig)

	// init mysql models
	models.Init()
	// init k8sConfig
	resource.InitK8sConfig()
	gitops.CloneClusterConfigRepo()

	controllermanager.Init()

	routers.Init()
}

func main() {
	beego.Info("Beego version:", beego.VERSION)
	beego.Info("Golang version:", runtime.Version())
	beego.Run()
}
