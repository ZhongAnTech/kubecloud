package controllers

import (
	"fmt"

	"github.com/astaxie/beego/orm"

	"kubecloud/backend/resource"
	"kubecloud/common"
	"kubecloud/common/utils"
)

type ClusterController struct {
	BaseController
}

func (cc *ClusterController) ClusterList() {
	filter := cc.GetFilterQuery()
	res, err := resource.GetClusterList(filter)
	if err != nil {
		cc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if res.Base.PageSize == 0 {
		// compatible old model
		cc.Data["json"] = NewResult(true, res.List, "")
	} else {
		cc.Data["json"] = NewResult(true, res, "")
	}
	cc.ServeJSON()
}

func (cc *ClusterController) InspectCluster() {
	clusterId := cc.GetStringFromPath(":cluster")

	result, err := resource.GetClusterDetail(clusterId)
	if err != nil {
		if err == orm.ErrNoRows {
			cc.ServeError(common.NewNotFound().SetCause(fmt.Errorf("database error: cluster(%s) is not existed!", clusterId)))
		} else if err == orm.ErrMultiRows {
			cc.ServeError(common.NewConflict().SetCause(fmt.Errorf("database error: cluster(%s) info is duplicated: %s!", clusterId, err.Error())))
		} else {
			cc.ServeError(common.NewInternalServerError().SetCause(err))
		}
		return
	}
	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ClusterController) CreateCluster() {
	var cluster resource.Cluster
	cc.DecodeJSONReq(&cluster)
	if cluster.ClusterId == "" {
		cluster.ClusterId = utils.NewUUID()
	}
	if err := cluster.Verify(); err != nil {
		cc.ServeError(common.NewBadRequest().SetCause(err))
		return
	}
	result, err := resource.CreateCluster(cluster)
	if err != nil {
		cc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ClusterController) DeleteCluster() {
	clusterId := cc.GetStringFromPath(":cluster")

	if err := resource.DeleteCluster(clusterId); err != nil {
		cc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	cc.Data["json"] = NewResult(true, nil, "")
	cc.ServeJSON()
}

func (cc *ClusterController) UpdateCluster() {
	clusterId := cc.GetStringFromPath(":cluster")

	var cluster resource.Cluster
	cc.DecodeJSONReq(&cluster)
	cluster.ClusterId = clusterId
	if err := cluster.Verify(); err != nil {
		cc.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	result, err := resource.UpdateCluster(cluster)
	if err != nil {
		cc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	cc.Data["json"] = NewResult(true, result, "")
	cc.ServeJSON()
}

func (cc *ClusterController) Certificate() {
	clusterId := cc.GetStringFromPath(":cluster")

	var cluster resource.Cluster
	cc.DecodeJSONReq(&cluster)
	if err := cluster.Verify(); err != nil {
		cc.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if err := resource.SetClusterCertificate(clusterId, cluster.Certificate); err != nil {
		cc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	cc.Data["json"] = NewResult(true, nil, "")
	cc.ServeJSON()
}
