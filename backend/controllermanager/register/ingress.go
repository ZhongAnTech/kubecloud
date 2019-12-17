package register

import (
	"fmt"
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/ingress"
)

func startIngressController(ctx cm.ControllerContext) error {
	ic, err := ingress.NewIngressController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Extensions().V1beta1().Ingresses())
	if err != nil {
		return fmt.Errorf("error creating ingress controller: %v", err)
	}
	go ic.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("ingress", startIngressController)
}
