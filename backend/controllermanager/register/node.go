package register

import (
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/node"
)

func startNodeController(ctx cm.ControllerContext) error {
	nc, err := node.NewNodeController(
		ctx.Cluster,
		ctx.InformerFactory.Core().V1().Nodes(),
		ctx.Client)
	if err != nil {
		return err
	}

	go nc.Run(1, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("node", startNodeController)
}
