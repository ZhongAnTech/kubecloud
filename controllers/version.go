package controllers

type VersionController struct {
	BaseController
}

type version struct {
	Version string `json:"version"`
}

const curVersion = "V0.1.0"

func (this *VersionController) Get() {
	ver := &version{
		Version: curVersion,
	}

	this.Data["json"] = NewResult(true, ver, "")
	this.ServeJSON()
}
