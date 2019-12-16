package resource

import (
	"encoding/json"
	"fmt"
	"testing"

	"kubecloud/backend/models"
	"kubecloud/common/utils"
	_ "kubecloud/test/mock"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var nativeTpl = `{
		"template":"apiVersion: v1\nkind: Service\nmetadata:\n  annotations:\n    creator: test\n    description: create from boom3\n    owner_name: native-test\n  creationTimestamp: null\n  labels:\n    app: native-test\n    version: 053e7ab6\n  name: svc-native-test\n  namespace: tech\nspec:\n  ports:\n  - name: http-8080-8080-8h9rn\n    port: 8080\n    protocol: TCP\n    targetPort: 8080\n  selector:\n    app: native-test\n  type: ClusterIP\nstatus:\n  loadBalancer: {}\n---\napiVersion: extensions/v1beta1\nkind: Ingress\nmetadata:\n  annotations:\n    created_default: \"true\"\n    creator: test\n    description: create from boom3\n    owner_name: native-test\n  creationTimestamp: null\n  labels:\n    app: native-test\n    version: 053e7ab6\n  name: ing-native-test\n  namespace: tech\nspec:\n  rules:\n  - host: native-test.dev.za-paas.net\n    http:\n      paths:\n      - backend:\n          serviceName: svc-native-test\n          servicePort: 8080\nstatus:\n  loadBalancer: {}\n---\napiVersion: apps/v1beta1\nkind: Deployment\nmetadata:\n  annotations:\n    creator: test\n    description: create from boom3\n    owner_name: native-test-053e7ab6\n    sidecar.istio.io/inject: \"false\"\n  creationTimestamp: null\n  labels:\n    app: native-test\n    version: 053e7ab6\n  name: native-test\n  namespace: tech\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: native-test\n      version: 053e7ab6\n  strategy:\n    type: Recreate\n  template:\n    metadata:\n      annotations:\n        creator: test\n        description: create from boom3\n        owner_name: native-test-053e7ab6\n        sidecar.istio.io/inject: \"false\"\n      creationTimestamp: null\n      labels:\n        app: native-test\n        version: 053e7ab6\n      name: native-test-053e7ab6\n      namespace: tech\n    spec:\n      containers:\n      - env:\n        - name: DEPLOY_ENV\n          value: test\n        - name: SERVICE_CHECK_HTTP\n          value: /health\n        - name: SERVICE_CHECK_INTERVAL\n          value: 30s\n        - name: SERVICE_CHECK_TIMEOUT\n          value: 8s\n        - name: PROJECT_ID\n          value: \"534\"\n        - name: SERVICE_NAME\n          value: 534-tech_other_za-helloworld-053e7ab6-1556447449\n        - name: SERVICE_TAGS\n          value: native-test/\n        - name: HOST_IP\n          valueFrom:\n            fieldRef:\n              apiVersion: v1\n              fieldPath: status.hostIP\n        image: 10.253.15.66:5050/zis/za-helloworld:v1\n        imagePullPolicy: Always\n        livenessProbe:\n          failureThreshold: 5\n          initialDelaySeconds: 30\n          periodSeconds: 60\n          successThreshold: 1\n          tcpSocket:\n            port: 8080\n          timeoutSeconds: 2\n        name: native-test\n        readinessProbe:\n          failureThreshold: 3\n          httpGet:\n            path: /health\n            port: 8080\n          initialDelaySeconds: 10\n          periodSeconds: 30\n          successThreshold: 1\n          timeoutSeconds: 10\n        resources:\n          limits:\n            cpu: \"1\"\n            memory: 1Gi\n          requests:\n            cpu: 100m\n            memory: 128Mi\n        securityContext:\n          privileged: false\n        volumeMounts:\n        - mountPath: /alidata1/admin/native-test/logs\n          name: native-test-volume-rmxdj\n        - mountPath: /etc/localtime\n          name: native-test-volume-pp682\n          readOnly: true\n      dnsPolicy: ClusterFirst\n      imagePullSecrets:\n      - name: harbor-dev\n      nodeSelector:\n        com.zhonganinfo.bizcluster: zis\n      restartPolicy: Always\n      volumes:\n      - hostPath:\n          path: /alidata1/admin/logs/za-helloworld\n        name: native-test-volume-rmxdj\n      - hostPath:\n          path: /etc/localtime\n        name: native-test-volume-pp682\nstatus: {}",
		"config":{
			"default_port":8080,
			"inject_service_mesh":false,
			"version":"v1",
			"deploy_strategy":"",
			"autoscaling":{
				"min_replicas":1,
				"max_replicas":3,
				"cpu_target":80,
				"memory_target":0
			}
		}
	}`

func getTestAppName() string {
	appname := "native-test"
	return appname
}

func getTemplate(tplkind, appkind string) (Template, error) {
	template := NewTemplate(tplkind)
	tpl := nativeTpl
	err := json.Unmarshal([]byte(tpl), template)
	if err != nil {
		return nil, err
	}
	return template, nil
}

func createTestPVC(ar *AppRes, namespace string) error {
	pvc := v1.PersistentVolumeClaim{}
	pvc.Name = "test"
	_, err := ar.Client.CoreV1().PersistentVolumeClaims(namespace).Get("test", metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		return err
	}
	_, err = ar.Client.CoreV1().PersistentVolumeClaims(namespace).Create(&pvc)
	return err
}

func installApp(ar *AppRes, namespace, tplkind, appkind string, param *ExtensionParam) error {
	projectid := int64(DEFAULT_PROJECT_ID)
	user := "test"
	template, err := getTemplate(tplkind, appkind)
	if err != nil {
		return err
	}

	return ar.InstallApp(projectid, namespace, "", template, param)
}

func TestInstallApp(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	ar, err := NewAppRes(cluster, nil)
	assert.Nil(t, err)
	t.Run("simple template application install/get/delete", func(t *testing.T) {
		appname := "simple-test"
		fmt.Println("install, no alert")
		err := installApp(ar, namespace, models.SIMPLE_TEMPLATE, AppKindDeployment, nil)
		assert.Nil(t, err)
		fmt.Println("install, has alert")
		eparam := ExtensionParam{}
		err = installApp(ar, namespace, models.SIMPLE_TEMPLATE, AppKindDeployment, &eparam)
		assert.Nil(t, err)
		fmt.Println("get simple-test detail")
		appdetail, err := ar.GetAppDetail(namespace, appname)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		assert.Equal(t, appname, appdetail.Name)
		fmt.Println("delete simple-test app")
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	})
	t.Run("native template application install/get/delete", func(t *testing.T) {
		appname := "native-test"
		fmt.Println("install, no alert ")
		err := installApp(ar, namespace, models.NATIVE_TEMPLATE, AppKindDeployment, nil)
		assert.Nil(t, err)
		fmt.Println("install, has alert ")
		eparam := ExtensionParam{}
		err = installApp(ar, namespace, models.NATIVE_TEMPLATE, AppKindDeployment, &eparam)
		assert.Nil(t, err)
		fmt.Println("get native-test detail")
		appdetail, err := ar.GetAppDetail(namespace, appname)
		assert.Nil(t, err)
		assert.Equal(t, appname, appdetail.Name)
		fmt.Println("delete native-test app")
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	})
}

func TestReconfigApp(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	ar, err := NewAppRes(cluster, nil)
	assert.Nil(t, err)
	testFunc := func(tplkind, appkind string) {
		appname := getTestAppName()
		err := installApp(ar, namespace, tplkind, appkind, nil)
		if err != nil {
			t.Fatalf("install app failed, %v", err)
		}
		app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
		if err != nil {
			t.Fatalf("get app by name failed: %v, %s", err, appname)
		}
		tpl, err := CreateAppTemplateByApp(*app)
		if err != nil {
			t.Fatalf("get app template failed, %v", err)
		}
		_, err = ar.ReconfigureApp(*app, tpl.Replicas(2))
		assert.Nil(t, err)
		app, err = ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
		if err != nil {
			t.Fatalf("get app by name failed: %v", err)
		}
		assert.Equal(t, 2, app.Replicas)
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	}
	t.Run("update native application reconfig", func(t *testing.T) {
		testFunc(models.NATIVE_TEMPLATE, AppKindDeployment)
	})
}

func TestRestartApp(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	ar, err := NewAppRes(cluster, nil)
	assert.Nil(t, err)
	testFunc := func(tplkind, appkind string) {
		appname := getTestAppName(tplkind, appkind)
		err := installApp(ar, namespace, tplkind, appkind, nil)
		if err != nil {
			t.Fatalf("install app failed, %v", err)
		}
		err = ar.Restart(namespace, appname)
		if err != nil {
			t.Fatalf("Restart app failed, %v, %s", err, appname)
		}
		assert.Nil(t, err)
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	}
	t.Run("restart native application reconfig", func(t *testing.T) {
		testFunc(models.NATIVE_TEMPLATE, AppKindDeployment)
	})
}

func TestRollingUpdateApp(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	image := "10.253.15.66:5050/zis/helloworld:v2"
	ar, err := NewAppRes(cluster, nil)
	assert.Nil(t, err)
	testFunc := func(cname, image, kind string) {
		appname := getTestAppName(kind, AppKindDeployment)
		err := installApp(ar, namespace, kind, AppKindDeployment, nil)
		if err != nil {
			t.Fatalf("install app failed, %v", err)
		}
		param := []ContainerParam{
			{
				Name:  cname,
				Image: image,
			},
		}
		app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
		if err != nil {
			t.Fatalf("get app by name failed: %v", err)
		}
		assert.NotEqual(t, image, app.Image)
		err = ar.RollingUpdateApp(namespace, appname, param)
		app, err = ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
		if err != nil {
			t.Fatalf("get app by name failed: %v", err)
		}
		assert.Equal(t, image, app.Image)
		assert.Nil(t, err)
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	}
	t.Run("rolling update simple application", func(t *testing.T) {
		testFunc(getTestAppName(models.SIMPLE_TEMPLATE, AppKindDeployment), image, models.SIMPLE_TEMPLATE)
	})
	t.Run("rolling update native application", func(t *testing.T) {
		testFunc(getTestAppName(models.NATIVE_TEMPLATE, AppKindDeployment), image, models.NATIVE_TEMPLATE)
	})
}

func TestScaleApp(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	ar, err := NewAppRes(cluster, nil)
	assert.Nil(t, err)
	testFunc := func(kind, appkind string) {
		appname := getTestAppName(kind, appkind)
		err := installApp(ar, namespace, kind, appkind, nil)
		if err != nil {
			t.Fatalf("install app failed, %v", err)
		}
		testScale := func(destNum int) {
			err = ar.ScaleApp(namespace, appname, 2)
			assert.Nil(t, err)
			app, err := ar.Appmodel.GetAppByName(ar.Cluster, namespace, appname)
			if err != nil {
				t.Fatalf("get app by name failed: %v", err)
			}
			assert.Equal(t, 2, app.Replicas)
		}
		testScale(2)
		testScale(1)
		err = ar.DeleteApp(namespace, appname)
		assert.Nil(t, err)
	}
	t.Run("scale simple deployment application", func(t *testing.T) {
		testFunc(models.SIMPLE_TEMPLATE, AppKindDeployment)
	})
}
func TestGetAppList(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	ar, err := NewAppRes(cluster, nil)
	if err != nil {
		t.Fatalf("get app resource handler faliled for %v", err)
	}
	if ar == nil {
		t.Fatalf("get app resource handler faliled!")
	}
	t.Run("AppsIsEmtpy", func(t *testing.T) {
		flist, err := ar.GetAppList("test", nil, nil)
		assert.Nil(t, err)
		assert.Empty(t, flist.List)
	})
	t.Run("Two Apps", func(t *testing.T) {
		err := installApp(ar, namespace, models.SIMPLE_TEMPLATE, AppKindDeployment, nil)
		if err != nil {
			t.Fatalf("install simple app failed, %v", err)
		}
		err = installApp(ar, namespace, models.NATIVE_TEMPLATE, AppKindDeployment, nil)
		if err != nil {
			t.Fatalf("install simple app failed, %v", err)
		}
		fmt.Println("filter is nil")
		flist, err := ar.GetAppList(namespace, []string{"native-test", "simple-test"}, nil)
		assert.Nil(t, err)
		applist, ok := flist.List.([]AppItem)
		assert.Equal(t, true, ok)
		assert.Equal(t, 2, len(applist))
		fmt.Println("filter is name")
		filterQuery := utils.FilterQuery{
			FilterKey: "name",
			FilterVal: "simple-test",
		}
		flist, err = ar.GetAppList(namespace, []string{"native-test", "simple-test"}, &filterQuery)
		assert.Nil(t, err)
		applist, ok = flist.List.([]AppItem)
		assert.Equal(t, true, ok)
		assert.Equal(t, 1, len(applist))
		if err = ar.DeleteApp(namespace, "native-test"); err != nil {
			t.Fatalf("delete app failed, %v", err)
		}
		if err = ar.DeleteApp(namespace, "simple-test"); err != nil {
			t.Fatalf("delete app failed, %v", err)
		}
	})
}
