package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/astaxie/beego"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/resource"
	"kubecloud/common"
	"kubecloud/common/validate"
)

type HarborController struct {
	BaseController
}

func (this *HarborController) HarborCreate() {
	var harborReq models.HarborReq
	this.DecodeJSONReq(&harborReq)
	beego.Debug("harbor create debug: ", harborReq)
	if err := harborReq.Verify(&harborReq); err != nil {
		this.ServeError(err)
		return
	}

	if err := resource.HarborCreate(&harborReq); err != nil {
		this.ServeError(err)
		return
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
	return
}

func (this *HarborController) HarborUpdate() {
	harbor := this.GetStringFromPath(":harbor")
	var harborReq models.HarborReq
	this.DecodeJSONReq(&harborReq)
	harborReq.HarborId = harbor
	if err := harborReq.Verify(&harborReq); err != nil {
		this.ServeError(err)
		return
	}

	if err := resource.HarborUpdate(&harborReq); err != nil {
		this.ServeError(err)
		return
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
	return
}

func (this *HarborController) HarborDelete() {
	harborId := this.GetStringFromPath(":harbor")
	if err := resource.HarborDelete(harborId); err != nil {
		this.ServeError(err)
		return
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
	return
}

func (this *HarborController) HarborList() {
	filterQuery := this.GetFilterQuery()
	res, err := resource.HarborList(filterQuery)
	if err != nil {
		this.ServeError(err)
		return
	}
	harbors, _ := res.List.([]models.ZcloudHarbor)
	res.List = harbors
	this.Data["json"] = NewResult(true, res, "")
	this.ServeJSON()
	return
}

func (this *HarborController) HarborInspect() {
	id := this.GetStringFromPath(":harbor")
	harbor, err := resource.HarborInspect(id)
	if err != nil {
		this.ServeError(err)
		return
	}
	this.Data["json"] = NewResult(true, harbor, "")
	this.ServeJSON()
	return
}

func (this *HarborController) RepositoriesList() {
	harbor := this.GetStringFromPath(":harbor")
	//project := this.GetString("project")
	public := this.GetString("public")

	res, err := dao.GetRepositoriesList(harbor, public, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, res.List, "")
	this.ServeJSON()
	return
}

func (this *HarborController) NormalRepositoriesList() {
	harbor := this.GetStringFromPath(":harbor")
	public := this.GetString("public")
	filterQuery := this.GetFilterQuery()
	res, err := dao.GetRepositoriesList(harbor, public, filterQuery)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, res, "")
	this.ServeJSON()
	return
}

func (this *HarborController) RepositoryDelete() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")

	harborInspect, err := resource.HarborInspect(harbor)
	if err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	harborClient := resource.NewHarborClient(harborInspect)

	if err := dao.DeleteRepository(harbor, repository); err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	tagList, err := harborClient.GetHarborRepositoryTags(repository)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	for _, tag := range tagList {
		if err := harborClient.DeleteHarborRepositoryTag(repository, tag.Tag); err != nil {
			this.ServeError(common.NewInternalServerError().SetCause(err))
			return
		}
	}
	if err := dao.DeleteRepositoryTags(harbor, repository); err != nil {
		beego.Warn("delete repository tags from database failed:", err)
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *HarborController) RepositoryTagsList() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")

	result, err := resource.HarborRepositoryTagList(harbor, repository, nil)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, result.List, "")
	this.ServeJSON()
}

func (this *HarborController) NormalRepositoryTagsList() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")
	filterQuery := this.GetFilterQuery()
	result, err := resource.HarborRepositoryTagList(harbor, repository, filterQuery)
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, result, "")
	this.ServeJSON()
}

func (this *HarborController) DeleteRepositoryTag() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")
	tag := this.GetStringFromPath(":tag")

	harborInspect, err := resource.HarborInspect(harbor)
	if err != nil {
		this.ServeError(err)
		return
	}
	harborClient := resource.NewHarborClient(harborInspect)

	image := fmt.Sprintf("%v/%v:%v", harborInspect.HarborAddr, repository, tag)
	count := dao.CountAppByImage(image)
	if count > 0 {
		this.ServeError(common.NewBadRequest().SetCode("HarborImageInUse").SetMessage(`harbor image in use`))
		return
	}

	if err := harborClient.DeleteHarborRepositoryTag(repository, tag); err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := dao.DeleteRepositoryTag(harbor, repository, tag); err != nil {
		beego.Warn("delete repository tag from database failed:", err)
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

func (this *HarborController) SetHarborProjectPublic() {
	harbor := this.GetStringFromPath(":harbor")
	project := this.GetStringFromPath(":project")
	public := this.GetStringFromPath(":public")

	harborInspect, err := resource.HarborInspect(harbor)
	if err != nil {
		this.ServeError(err)
		return
	}
	harborClient := resource.NewHarborClient(harborInspect)

	publicInt, err := strconv.Atoi(public)
	if err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if publicInt != 0 && publicInt != 1 {
		err := fmt.Errorf("parameter can only be 0 or 1")
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if err := validate.IsReservedBuName(project); err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if err := harborClient.SetHarborProjectPublic(project, publicInt); err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}

type TagLabels struct {
	Labels map[string]string `json:"labels"`
}

func (this *HarborController) GetRepositoryTagLabels() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")
	tag := this.GetStringFromPath(":tag")

	tagDetail, err := dao.GetRepositoryTagDeatil(harbor, repository, tag)
	if err != nil {
		harborInspect, err := resource.HarborInspect(harbor)
		if err != nil {
			this.ServeError(common.NewInternalServerError().SetCause(err))
			return
		}
		harborClient := resource.NewHarborClient(harborInspect)
		if harborClient.RepositoryTagExist(repository, tag) {
			if err := dao.SetRepositoryTagLabels(harbor, repository, tag, ""); err != nil {
				this.ServeError(common.NewInternalServerError().SetCause(err))
				return
			}
			tagDetail, err = dao.GetRepositoryTagDeatil(harbor, repository, tag)
			if err != nil {
				this.ServeError(common.NewInternalServerError().SetCause(err))
				return
			}
		} else {
			this.ServeError(common.NewInternalServerError().SetCause(err))
			return
		}
	}

	labels := TagLabels{
		Labels: map[string]string{},
	}
	if tagDetail.Labels != "" {
		if err := json.Unmarshal([]byte(tagDetail.Labels), &labels.Labels); err != nil {
			this.ServeError(common.NewBadRequest().SetCause(err))
			return
		}
	}

	this.Data["json"] = NewResult(true, labels, "")
	this.ServeJSON()
}

func (this *HarborController) SetRepositoryTagLabels() {
	harbor := this.GetStringFromPath(":harbor")
	repository := this.GetStringFromPath(":splat")
	tag := this.GetStringFromPath(":tag")

	var labels TagLabels
	this.DecodeJSONReq(&labels)

	labelsStr, err := json.Marshal(&labels.Labels)
	if err != nil {
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if err := dao.SetRepositoryTagLabels(harbor, repository, tag, string(labelsStr)); err != nil {
		this.ServeError(err)
		return
	}

	this.Data["json"] = NewResult(true, nil, "")
	this.ServeJSON()
}
