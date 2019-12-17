package register

import (
	cm "kubecloud/backend/controllermanager"
	"kubecloud/backend/controllers/event"
)

func startEventController(ctx cm.ControllerContext) error {
	ec := event.NewEventController(
		ctx.Cluster,
		ctx.InformerFactory.Core().V1().Events(),
		ctx.Client)

	go ec.Run(1, ctx.Stop)
	return nil
}

func init() {
	cm.RegisterController("event", startEventController)
}
