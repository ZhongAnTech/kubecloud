package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"kubecloud/common"
	"kubecloud/common/utils"
)

// BaseController wraps common methods for controllers to host API
type BaseController struct {
	beego.Controller
	preparedAt time.Time
}

// GetStringFromPath gets the param from path and returns it as string
func (b *BaseController) GetStringFromPath(key string) string {
	return b.Ctx.Input.Param(key)
}

// GetInt64FromPath gets the param from path and returns it as int64
func (b *BaseController) GetInt64FromPath(key string) (int64, error) {
	value := b.Ctx.Input.Param(key)
	return strconv.ParseInt(value, 10, 64)
}

// GetInt64FromQuery gets the param from query string and returns it as int64
func (b *BaseController) GetInt64FromQuery(key string) (int64, error) {
	value := b.Ctx.Input.Query(key)
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		convErr := err.(*strconv.NumError)
		if convErr.Num == "" {
			return -1, nil
		} else {
			return v, err
		}
	}

	return v, nil
}

// GetBoolFromQuery gets the param from query string and returns it as int64
func (b *BaseController) GetBoolFromQuery(key string) (bool, error) {
	value := b.Ctx.Input.Query(key)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return boolValue, err
	}

	return boolValue, nil
}

// GetStringFromQuery gets the param from query string and returns it as string
func (b *BaseController) GetStringFromQuery(key string) string {
	return b.Ctx.Input.Query(key)
}

// GetPagination get pagination from query strings
func (ctrl *BaseController) GetPagination(defaultOffset, defaultLimit int) (*utils.Pagination, *common.Error) {
	parse := func(key string, defaultValue int) (int, error) {
		str := ctrl.Ctx.Input.Query(key)
		if str == "" {
			return defaultValue, nil
		}
		num, err := strconv.Atoi(str)
		if err != nil {
			return 0, err
		}
		if num < 0 {
			return 0, fmt.Errorf(`"%v" should be greater than 0`, key)
		}
		return num, nil
	}
	offset, err := parse("page_index", defaultOffset)
	if err != nil {
		return nil, common.NewBadRequest().SetCode("InvalidPaginationOffset").SetMessage("invalid pagination offset").SetCause(err)
	}
	limit, err := parse("page_size", defaultLimit)
	if err != nil {
		return nil, common.NewBadRequest().SetCode("InvalidPaginationLimit").SetMessage("invalid pagination limit").SetCause(err)
	}

	return &utils.Pagination{
		Offset: offset,
		Limit:  limit,
	}, nil
}

// Render returns nil as it won't render template
func (b *BaseController) Render() error {
	return nil
}

// RenderError provides shortcut to render http error
func (b *BaseController) RenderError(code int, text string) {
	http.Error(b.Ctx.ResponseWriter, text, code)
}

// DecodeJSONReq decodes a json request
func (b *BaseController) DecodeJSONReq(v interface{}) {
	err := json.Unmarshal(b.Ctx.Input.CopyBody(1<<32), v)
	if err != nil {
		beego.Error("Invalid json request: " + err.Error())
		b.CustomAbort(http.StatusBadRequest, "Invalid json request: "+err.Error())
	}
}

// GetReqBody returns body in string format
func (b *BaseController) GetReqBody() []byte {
	return b.Ctx.Input.CopyBody(1 << 32)
}

// Validate validates v if it implements interface validation.ValidFormer
func (b *BaseController) Validate(v interface{}) {
	validator := validation.Validation{}
	isValid, err := validator.Valid(v)
	if err != nil {
		b.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	if !isValid {
		message := ""
		for _, e := range validator.Errors {
			message += fmt.Sprintf("%s %s \n", e.Field, e.Message)
		}
		b.CustomAbort(http.StatusBadRequest, message)
	}
}

// DecodeJSONReqAndValidate does both decoding and validation
func (b *BaseController) DecodeJSONReqAndValidate(v interface{}) {
	b.DecodeJSONReq(v)
	b.Validate(v)
}

// Get page filter query
func (b *BaseController) GetFilterQuery() *utils.FilterQuery {
	filter := utils.FilterQuery{
		IsLike:    true,
		PageSize:  utils.DEF_PAGE_SIZE,
		PageIndex: utils.DEF_PAGE_INDEX,
	}
	b.DecodeJSONReq(&filter)

	return &filter
}

// SetResponseTime set reponse time
func (c *BaseController) SetResponseTime() {
	duration := time.Since(c.preparedAt)
	c.Ctx.Output.Header("X-Response-Time", duration.Truncate(time.Millisecond).String())
}

// SetStatus set status code
func (c *BaseController) SetStatus(status int) {
	c.Ctx.Output.SetStatus(status)
}

func (c *BaseController) ServeJSON(encoding ...bool) {
	c.SetResponseTime()
	c.Controller.ServeJSON(encoding...)
}

// ServeResult serve result, status code is 200
func (b *BaseController) ServeResult(result *Result) {
	b.Data["json"] = result
	b.ServeJSON()
}

// ServeError serve error
func (c *BaseController) ServeError(err error) {
	if err == nil {
		err = fmt.Errorf("nil")
	}
	var statusCode int
	var result *Result
	switch srcErr := err.(type) {
	// zcloud error
	case *common.Error:
		{
			errDetail := ""
			if srcErr.Cause() != nil {
				errDetail = srcErr.Cause().Error()
			}
			statusCode = srcErr.Status()
			result = NewErrorResult(srcErr.Code(), srcErr.Message(), errDetail)
		}
	// k8s error
	case apierrors.APIStatus:
		{
			apiStatus := srcErr.Status()
			statusCode = int(apiStatus.Code)
			result = NewErrorResult("KubernetesError", "kubernetes error", apiStatus.Message)
		}
	// go error
	default:
		{
			statusCode = http.StatusInternalServerError
			result = NewErrorResult("InternalServerError", "internal server error", err.Error())
		}
	}
	if result != nil {
		result.Origin = "zcloud"
	}
	c.Ctx.Output.SetStatus(statusCode)
	c.ServeResult(result)
}
