package register

import (
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/harbor"
)

func startHarborController(ctx cm.ControllerContext) error {
	hc, err := harbor.NewHarborController(ctx.Cluster)
	if err != nil {
		return err
	}
	go hc.Run(ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("harbor", startHarborController)
}
