package controllers

import (
	"kubecloud/backend/resource"
	"kubecloud/common"
)

type NodeController struct {
	BaseController
}

func (nc *NodeController) ListNode() {
	cluster := nc.GetStringFromPath(":cluster")

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := node.ListNode()
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}

func (nc *NodeController) GetNode() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := node.GetNode(nodeName)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeUpdate() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	var nodeUpdate map[string]string
	nc.DecodeJSONReq(&nodeUpdate)

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := node.UpdateNode(nodeName, nodeUpdate); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeDelete() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := node.DeleteNode(nodeName); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeFreeze() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")
	deletePods := nc.GetStringFromQuery("deletePods")

	var delete bool
	if deletePods == "true" {
		delete = true
	}

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := node.FreezeNode(nodeName, delete); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeUnfreeze() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	if err := node.UnfreezeNode(nodeName); err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, nil, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodePods() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := node.GetNodePods(nodeName)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}

func (nc *NodeController) NodeEvent() {
	cluster := nc.GetStringFromPath(":cluster")
	nodeName := nc.GetStringFromPath(":node")
	filter := nc.GetFilterQuery()

	node, err := resource.NewNode(cluster)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	result, err := node.GetNodeEvent(nodeName, filter)
	if err != nil {
		nc.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	nc.Data["json"] = NewResult(true, result, "")
	nc.ServeJSON()
}
