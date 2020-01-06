package routers

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/logs"

	"kubecloud/controllers"
)

func Init() {
	kubecloudAPI :=
		beego.NewNamespace("kubecloud/api",
			beego.NSNamespace("/v1",
				beego.NSRouter("/events", &controllers.EventsController{}, "get:Get"),
				beego.NSRouter("/version", &controllers.VersionController{}, "get:Get"),

				// cluster
				beego.NSRouter("/clusters", &controllers.ClusterController{}, "post:CreateCluster"),
				beego.NSRouter("/clusters/list", &controllers.ClusterController{}, "post:ClusterList"),
				beego.NSRouter("/clusters/:cluster", &controllers.ClusterController{}, "get:InspectCluster;put:UpdateCluster;delete:DeleteCluster"),

				// node
				beego.NSRouter("/clusters/:cluster/nodes", &controllers.NodeController{}, "post:ListNode"),
				beego.NSRouter("/clusters/:cluster/nodes/:node", &controllers.NodeController{}, "get:GetNode;put:NodeUpdate;delete:NodeDelete"),
				beego.NSRouter("/clusters/:cluster/nodes/:node/freeze", &controllers.NodeController{}, "post:NodeFreeze"),
				beego.NSRouter("/clusters/:cluster/nodes/:node/unfreeze", &controllers.NodeController{}, "post:NodeUnfreeze"),
				beego.NSRouter("/clusters/:cluster/nodes/:node/pods", &controllers.NodeController{}, "get:NodePods"),
				beego.NSRouter("/clusters/:cluster/nodes/:node/event", &controllers.NodeController{}, "post:NodeEvent"),

				// namespace
				beego.NSRouter("/clusters/:cluster/namespaces", &controllers.NamespaceController{}, "post:Create"),
				beego.NSRouter("/clusters/:cluster/namespaces/list", &controllers.NamespaceController{}, "post:NamespaceList"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace", &controllers.NamespaceController{}, "get:Inspect;put:Update;delete:Delete"),

				// app
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps", &controllers.AppController{}, "post:Create"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/label", &controllers.AppController{}, "post:SetAppLabels"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/rollingupdate", &controllers.AppController{}, "post:BatchRollingUpdate"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/list", &controllers.AppController{}, "post:List"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app", &controllers.AppController{}, "get:Inspect;delete:Delete"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/restart", &controllers.AppController{}, "post:Restart"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/reconfigure", &controllers.AppController{}, "post:Reconfigure"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/rollingupdate", &controllers.AppController{}, "post:RollingUpdate"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/scale", &controllers.AppController{}, "post:Scale"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/log", &controllers.AppController{}, "get:Log"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/event", &controllers.AppController{}, "get:Event"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/pods/:podname/status", &controllers.AppController{}, "get:PodInspect"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/pods/:podname/containernames/:containername/terminal", &controllers.TermController{}, "get:PodTerminal"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/apps/:app/pods/:podname/containernames/:containername/exec", &controllers.ExecController{}, "get:PodTerminalExec"),

				// secret
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/secrets", &controllers.SecretController{}, "post:Create"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/secrets/list", &controllers.SecretController{}, "post:List"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/secrets/:secret", &controllers.SecretController{}, "delete:Delete;put:Update"),

				// ingress
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/ingresses", &controllers.IngressController{}, "post:Create"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/ingresses/list", &controllers.IngressController{}, "post:List"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/ingresses/:ingressID", &controllers.IngressController{}, "delete:Delete;put:Update;get:Inspect"),

				// service
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/services", &controllers.ServiceController{}, "get:List"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/services/:service", &controllers.ServiceController{}, "get:Inspect"),

				// harbor
				beego.NSRouter("/harbors", &controllers.HarborController{}, "post:HarborCreate"),
				beego.NSRouter("/harbors/list", &controllers.HarborController{}, "post:HarborList"),
				beego.NSRouter("/harbors/:harbor", &controllers.HarborController{}, "get:HarborInspect;put:HarborUpdate;delete:HarborDelete"),
				beego.NSRouter("/harbors/:harbor/repositories", &controllers.HarborController{}, "get:RepositoriesList"),
				beego.NSRouter("/harbors/:harbor/repositories/list", &controllers.HarborController{}, "post:NormalRepositoriesList"),
				beego.NSRouter("/harbors/:harbor/repositories/*", &controllers.HarborController{}, "delete:RepositoryDelete"),
				beego.NSRouter("/harbors/:harbor/repositories/*/tags", &controllers.HarborController{}, "get:RepositoryTagsList"),
				beego.NSRouter("/harbors/:harbor/repositories/*/tags/list", &controllers.HarborController{}, "post:NormalRepositoryTagsList"),
				beego.NSRouter("/harbors/:harbor/repositories/*/tags/:tag", &controllers.HarborController{}, "delete:DeleteRepositoryTag"),
				beego.NSRouter("/harbors/:harbor/repositories/*/tags/:tag/labels", &controllers.HarborController{}, "get:GetRepositoryTagLabels;post:SetRepositoryTagLabels"),
				beego.NSRouter("/harbors/:harbor/projects/:project/public/:public", &controllers.HarborController{}, "post:SetHarborProjectPublic"),

				// Configmap
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/configmaps", &controllers.ConfigMapController{}, "get:List;post:Create"),
				beego.NSRouter("/clusters/:cluster/namespaces/:namespace/configmaps/:configmap", &controllers.ConfigMapController{}, "get:Inspect;put:Update;delete:Delete"),
			),
		)

	beego.AddNamespace(kubecloudAPI)

	beego.SetStaticPath("/apidoc", "./apidoc")

	beego.Get("/health", func(ctx *context.Context) {
		ctx.Output.Body([]byte("OK"))
	})

	beego.Handler("/api/v3/socket/:info(.*)", controllers.CreateAttachHandler("/socket"))

	beego.ErrorController(&controllers.ErrorController{})

	// setup panic recover
	beego.BConfig.RecoverFunc = func(ctx *context.Context) {
		err := recover()
		if err == nil {
			return
		}
		if err == beego.ErrAbort {
			return
		}
		logs.Critical("The request is:", ctx.Input.Method(), ctx.Input.URL())
		logs.Critical("Handler crashed error:", err)
		var stack string
		for i := 1; ; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break
			}
			logs.Critical(fmt.Sprintf("%s:%d", file, line))
			stack = stack + fmt.Sprintln(fmt.Sprintf("%s:%d", file, line))
		}
		hasIndent := beego.BConfig.RunMode != beego.PROD
		result := controllers.NewErrorResult("Panic", fmt.Sprintf("%v", err), stack)
		ctx.Output.SetStatus(http.StatusInternalServerError)
		ctx.Output.JSON(result, hasIndent, false)
	}
}
