package controllers

import (
	"kubecloud/backend/resource"
	"kubecloud/common"
)

type NodeController struct {
	BaseController
}

func (nc *NodeController) NodeList() {
	clusterId := nc.GetStringFromPath(":cluster")
	filter := nc.GetFilterQuery()
	result, err := resource.GetNodeListFilter(clusterId, filter)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if result.Base.PageSize == 0 {
		// compatible old model
		nc.Data["json"] = NewResult(true, result.List, "")
	} else {
		nc.Data["json"] = NewResult(true, result, "")
	}
	nc.ServeJSON()
}

func (nc *NodeController) NodeInspect() {
	clusterId := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	result, err := resource.GetNodeDetail(clusterId, nodeName)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeUpdate() {
	clusterId := nc.GetStringFromPath(":cluster")
	node := nc.GetStringFromPath(":node")

	var nodeUpdate resource.NodeUpdate
	nc.DecodeJSONReq(&nodeUpdate)
	nodeUpdate.Cluster = clusterId
	if err := nodeUpdate.Verify(); err != nil {
		nc.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	if err := resource.UpdateNode(clusterId, node, nodeUpdate); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeDelete() {
	clusterId := nc.GetStringFromPath(":cluster")
	node := nc.GetStringFromPath(":node")

	if err := resource.DeleteNode(clusterId, node); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeFreeze() {
	clusterId := nc.GetStringFromPath(":cluster")
	node := nc.GetStringFromPath(":node")

	var nodeFreeze resource.NodeFreeze
	nc.DecodeJSONReq(&nodeFreeze)

	if err := resource.FreezeNode(clusterId, node, nodeFreeze.DeletePods); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeUnfreeze() {
	clusterId := nc.GetStringFromPath(":cluster")
	node := nc.GetStringFromPath(":node")

	if err := resource.UnfreezeNode(clusterId, node); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodePods() {
	clusterId := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	result, err := resource.GetNodePods(clusterId, nodeName)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeEvent() {
	filter := nc.GetFilterQuery()
	clusterId := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	result, err := resource.GetNodeEvent(clusterId, nodeName, filter)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}
