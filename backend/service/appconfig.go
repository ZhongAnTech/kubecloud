package service

import "github.com/astaxie/beego/config"

func GetAppConfig() config.Configer {
	return appConfigProvider()
}
