package resource

import (
	"fmt"
	"net/http"

	"github.com/astaxie/beego/orm"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/common"
	"kubecloud/common/utils"
)

type TemplateInfo struct {
	models.ZcloudTemplate
	CreateAt string `json:"create_at"`
	UpdateAt string `json:"update_at"`
	DeleteAt string `json:"delete_at"`
}

type TemplateRes struct {
	modelHandle *dao.TemplateModel
	listNSFunc  NamespaceListFunction
}

func NewTemplateRes(get NamespaceListFunction) *TemplateRes {
	return &TemplateRes{
		modelHandle: dao.NewTemplateModel(),
		listNSFunc:  get,
	}
}

// template interface, nativetemplate support this interface
type Template interface {
	Default(cluster string) Template
	Validate() error
	GetExample() []byte
	Deploy(projectid int64, cluster, namespace, tname string, eparam *ExtensionParam) error
}

func NewTemplate() Template {
	return NewNativeTemplate()
}

func (tr *TemplateRes) CreateTemplate(template models.ZcloudTemplate) (*models.ZcloudTemplate, error) {
	texist, err := tr.modelHandle.GetTemplate(template.Namespace, template.Name)
	if texist != nil {
		return nil, common.NewConflict().SetCode("TemplateAlreadyExists").SetMessage("template already exists")
	} else {
		if err != nil {
			if err != orm.ErrNoRows {
				return nil, common.NewInternalServerError().SetCause(err)
			}
		}
	}
	temp, err := tr.modelHandle.CreateTemplate(template)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}

	return temp, nil
}

func (tr *TemplateRes) DeleteTemplate(namespace, name string) error {
	_, err := tr.modelHandle.GetTemplate(namespace, name)
	if err != nil {
		if err == orm.ErrNoRows {
			return common.NewNotFound().SetCause(err)
		} else {
			return common.NewInternalServerError().SetCause(err)
		}
	}

	//todo if template is used by some apps, it can not be deleted
	err = tr.modelHandle.DeleteTemplate(namespace, name)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	return nil
}

func (tr *TemplateRes) UpdateTemplate(template models.ZcloudTemplate) (*models.ZcloudTemplate, error) {
	told, err := tr.modelHandle.GetTemplate(template.Namespace, template.Name)
	if err != nil {
		if err == orm.ErrNoRows {
			return nil, common.NewNotFound().SetCause(err)
		} else {
			return nil, common.NewInternalServerError().SetCause(err)
		}
	}
	told.Kind = template.Kind
	told.Spec = template.Spec
	told.Description = template.Description

	err = tr.modelHandle.UpdateTemplate(*told)
	if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}

	return told, nil
}

func (tr *TemplateRes) GetTemplateList(namespace string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	nslist := []string{}
	if namespace != common.AllNamespace {
		nslist = append(nslist, namespace)
	} else {
		nslist = tr.listNSFunc()
	}
	res, err := tr.modelHandle.GetTemplateList(nslist, filterQuery)
	if err != nil {
		if err == orm.ErrNoRows {
			return nil, common.NewNotFound().SetCause(err)
		}
		return nil, common.NewInternalServerError().SetCause(err)
	}
	tList, ok := res.List.([]models.ZcloudTemplate)
	if !ok {
		return nil, common.NewInternalServerError().SetCause(fmt.Errorf("invalid data type"))
	}
	list := []TemplateInfo{}
	for _, item := range tList {
		temp := TemplateInfo{}
		temp.ZcloudTemplate = item
		temp.UpdateAt = item.UpdateAt.Format("2006-01-02 15:04:05")
		temp.CreateAt = item.CreateAt.Format("2006-01-02 15:04:05")
		list = append(list, temp)
	}
	res.List = list
	return res, nil
}

func (tr *TemplateRes) GetTemplateByName(namespace, name string) (*models.ZcloudTemplate, int, error) {
	template, err := tr.modelHandle.GetTemplate(namespace, name)
	if err != nil {
		if err == orm.ErrNoRows {
			return template, http.StatusNotFound, err
		}
		return template, http.StatusInternalServerError, err
	}

	return template, http.StatusOK, nil
}

func (tr *TemplateRes) GetTemplateByID(id int64) (*models.ZcloudTemplate, int, error) {
	template, err := tr.modelHandle.GetTemplateByID(id)
	if err != nil {
		if err == orm.ErrNoRows {
			return template, http.StatusNotFound, err
		}
		return template, http.StatusInternalServerError, err
	}

	return template, http.StatusOK, nil
}
