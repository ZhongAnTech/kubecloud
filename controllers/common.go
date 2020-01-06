package controllers

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
