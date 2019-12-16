package register

import (
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/resourcequota"
)

func startResourceQuotaController(ctx cm.ControllerContext) error {
	controller := resourcequota.NewResourceQuotaController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Core().V1().ResourceQuotas())
	go controller.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	//cm.RegisterController("resourcequota", startResourceQuotaController)
}
