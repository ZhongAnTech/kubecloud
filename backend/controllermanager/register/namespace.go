package register

import (
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/namespace"
)

func startNamespaceController(ctx cm.ControllerContext) error {
	controller := namespace.NewNamespaceController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Core().V1().Namespaces())
	go controller.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	//cm.RegisterController("namespace", startNamespaceController)
}
