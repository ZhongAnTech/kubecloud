package register

import (
	"fmt"
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/application"
)

func startApplicationController(ctx cm.ControllerContext) error {
	ac, err := application.NewApplicationController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Apps().V1beta1().Deployments())
	if err != nil {
		return fmt.Errorf("error creating deployment controller: %v", err)
	}
	go ac.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("application", startApplicationController)
}
