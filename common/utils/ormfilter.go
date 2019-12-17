package utils

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"kubecloud/common/validate"
	"reflect"
	"strings"
)

type FilterContext struct {
	FilterKey string      `json:"filter_key"`
	FilterVal interface{} `json:"filter_val"`
	Operator  string      `json:"operator"`
}

type FilterQuery struct {
	PageIndex     int `json:"page_index"`
	PageSize      int `json:"page_size"`
	FilterContext `json:",inline"`
	IsLike        bool        `json:"is_like"`
	RequestBody   interface{} `json:"request_body"`
}

type DefaultFilter struct {
	Filters []FilterContext
}

const (
	FilterOperatorIn      = "in"
	FilterOperatorEqual   = ""
	FilterOperatorExclude = "not"

	DEF_PAGE_INDEX = 1
	DEF_PAGE_SIZE  = 10
)

func NewFilterQuery(isLike bool) *FilterQuery {
	return &FilterQuery{IsLike: isLike}
}

func (filter *FilterQuery) FilterCondition(filterKeys []string) *orm.Condition {
	filterSuffix := "icontains"
	cond := orm.NewCondition()
	if filter == nil {
		return nil
	}
	var items []string
	v := reflect.ValueOf(filter.FilterVal)
	switch v.Kind() {
	case reflect.String:
		val := filter.FilterVal.(string)
		if val == "" {
			return nil
		}
		items = strings.Split(val, ";")
		if validate.IsChineseChar(val) {
			filterSuffix = "contains"
		}
	case reflect.Int, reflect.Int64:
		if v.Int() == 0 {
			return nil
		}
	default:
		return nil
	}
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	if filter.IsLike {
		for _, key := range filterKeys {
			if len(items) > 1 {
				for _, item := range items {
					cond = cond.Or(fmt.Sprintf("%s__%s", key, filterSuffix), item)
				}
			} else {
				cond = cond.Or(fmt.Sprintf("%s__%s", key, filterSuffix), filter.FilterVal)
			}
		}
	} else {
		if filter.FilterKey != "" {
			if len(items) > 1 {
				cond = cond.And(filter.FilterKey+"__in", items)
			} else {
				cond = cond.And(filter.FilterKey, filter.FilterVal)
			}
		} else {
			for _, key := range filterKeys {
				if len(items) > 1 {
					for _, item := range items {
						cond = cond.Or(key, item)
					}
				} else {
					cond = cond.Or(key, filter.FilterVal)
				}
			}
		}
	}
	if cond.IsEmpty() {
		return nil
	}
	return cond
}

func (filter *FilterQuery) SetFilter(key string, val interface{}, operator string) *FilterQuery {
	filter.FilterVal = val
	filter.FilterKey = key
	filter.Operator = operator
	return filter
}

func NewDefaultFilter() *DefaultFilter {
	return &DefaultFilter{}
}

func (filter *DefaultFilter) DefaultFilterCondition() *orm.Condition {
	if filter == nil {
		return nil
	}
	cond := orm.NewCondition()
	for _, f := range filter.Filters {
		v := reflect.ValueOf(f.FilterVal)
		switch f.Operator {
		case FilterOperatorIn:
			if (v.Kind() == reflect.Array || v.Kind() == reflect.Slice) && v.Len() != 0 {
				cond = cond.And(f.FilterKey+"__in", f.FilterVal)
			}
		case FilterOperatorEqual:
			switch v.Kind() {
			case reflect.String:
				if f.FilterVal.(string) != "" {
					cond = cond.And(f.FilterKey, f.FilterVal)
				}
			default:
				cond = cond.And(f.FilterKey, f.FilterVal)
			}
		case FilterOperatorExclude:
			switch v.Kind() {
			case reflect.String:
				if f.FilterVal.(string) != "" {
					cond = cond.AndNot(f.FilterKey, f.FilterVal)
				}
			default:
				cond = cond.AndNot(f.FilterKey, f.FilterVal)
			}
		}
	}
	if cond.IsEmpty() {
		return nil
	}
	return cond
}

func (filter *DefaultFilter) AppendFilter(key string, val interface{}, op string) *DefaultFilter {
	filter.Filters = append(filter.Filters, FilterContext{FilterKey: key, FilterVal: val, Operator: op})
	return filter
}
