package controllermanager

type StartController func(ctx ControllerContext) error

var controllerList map[string]StartController

func RegisterController(name string, startFunc StartController) {
	if controllerList == nil {
		controllerList = make(map[string]StartController)
	}
	controllerList[name] = startFunc
}

func GetControllerList() map[string]StartController {
	return controllerList
}
