package keyword

const (
	LABEL_PRIMEDEPARTENT_ID_KEY = "prime_department_id"
	LABEL_SUBDEPARTENT_ID_KEY   = "sub_department_id"
	LABEL_APPNAME_KEY           = "app"
	LABEL_APPVERSION_KEY        = "appversion"
	LABEL_PODVERSION_KEY        = "version"

	ISTIO_INJECTION_POLICY  = "istio-injection"
	ISTIO_INJECTION_ENABLE  = "enabled"
	ISTIO_INJECTION_DISABLE = "disabled"

	RESTART_LABLE       = "kubecloud/restart"
	RESTART_LABLE_VALUE = "true"
	DELETE_LABLE        = "kubecloud/delete"
	DELETE_LABLE_VALUE  = "true"
	ENV_TEST_PUBLIC     = "test_public"
	ENV_DEV             = "dev"

	K8S_RESOURCE_TYPE_NODE = "node"
	K8S_RESOURCE_TYPE_APP  = "app"
)
