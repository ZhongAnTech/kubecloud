package register

import (
	"fmt"
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/secret"
)

func startSecretController(ctx cm.ControllerContext) error {
	sc, err := secret.NewSecretController(
		ctx.Cluster, ctx.Client,
		ctx.InformerFactory.Core().V1().Secrets())
	if err != nil {
		return fmt.Errorf("error creating secret controller: %v", err)
	}
	go sc.Run(ctx.Option.NormalConcurrentSyncs, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("secret", startSecretController)
}
