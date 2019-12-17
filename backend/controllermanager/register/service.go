package register

import (
	"fmt"
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/service"
)

func startServiceController(ctx cm.ControllerContext) error {
	ac, err := service.NewServiceController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Core().V1().Services())
	if err != nil {
		return fmt.Errorf("error creating service controller: %v", err)
	}
	go ac.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("service", startServiceController)
}
