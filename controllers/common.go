package controllers

import (
	"kubecloud/backend/dao"
	"kubecloud/common"
)

// TODO: deprecate IsSuccess, Origin
type Result struct {
	IsSuccess bool        `json:"IsSuccess"`
	Data      interface{} `json:"Data,omitempty"`
	ErrCode   string      `json:"ErrCode,omitempty"`
	ErrMsg    string      `json:"ErrMsg,omitempty"`
	ErrDetail string      `json:"ErrDetail,omitempty"`
	Origin    string      `json:"Origin,omitempty"`
}

func NewResult(isSuccess bool, data interface{}, errMsg string) *Result {
	return &Result{IsSuccess: isSuccess, Data: data, ErrMsg: errMsg}
}

func NewErrorResult(errCode, errMsg, errDetail string) *Result {
	return &Result{
		IsSuccess: false,
		ErrCode:   errCode,
		ErrMsg:    errMsg,
		ErrDetail: errDetail,
	}
}

//NamespaceListFunc ...
func NamespaceListFunc(cluster string, initns string) func() []string {
	return func() []string {
		nsList := []string{}
		if initns == common.AllNamespace {
			nsList, _ = dao.GetClusterNamespaceList(cluster)
		} else {
			nsList = append(nsList, initns)
		}
		return nsList
	}
}
