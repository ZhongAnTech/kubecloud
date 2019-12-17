package controllers

import (
	"kubecloud/common"
)

type ErrorController struct {
	BaseController
}

func (this *ErrorController) Error404() {
	err := common.NewNotFound()
	this.ServeError(err)
}

func (this *ErrorController) Error405() {
	err := common.NewMethodNotAllowed()
	this.ServeError(err)
}

func (this *ErrorController) Error501() {
	err := common.NewNotImplemented()
	this.ServeError(err)
}
