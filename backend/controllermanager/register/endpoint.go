package register

import (
	"fmt"
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/endpoint"
)

func startEndpointController(ctx cm.ControllerContext) error {
	ac, err := endpoint.NewEndpointController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Core().V1().Endpoints())
	if err != nil {
		return fmt.Errorf("error creating service controller: %v", err)
	}
	go ac.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("endpoint", startEndpointController)
}
