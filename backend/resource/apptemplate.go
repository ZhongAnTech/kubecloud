package resource

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"kubecloud/backend/models"

	"github.com/astaxie/beego"
	yamlencoder "github.com/ghodss/yaml"
)

const TemplateMaxSize = 1024000 //1000KB

//single app template interface
//simpleapp and nativeapp template support this interface
type AppTemplate interface {
	GenerateAppObject(cluster, namespace, tplname, domainSuffix string) (*models.ZcloudApplication, error)
	UpdateAppObject(app *models.ZcloudApplication, domainSuffix string) error
	GenerateKubeObject(cluster, namespace, podVersion, domainSuffix string) (map[string]interface{}, error)
	GetAppName() string
	GetAppKind() string
	GetAppVersion() string
	String() (string, error)
	Image(param []ContainerParam) AppTemplate
	DefaultLabel() AppTemplate
	Replicas(replicas int) AppTemplate
	IsInjectServiceMesh() bool
}

type WorkerResult struct {
	AppName    string
	AppVersion string
	AppKind    string
	Result     error
}

func CreateAppTemplateByApp(app models.ZcloudApplication) (AppTemplate, error) {
	return CreateNativeAppTemplate(app, "", nil)
}

func DeployAppTemplates(appTplList []AppTemplate, projectid int64, cluster, namespace, tname string, eparam *ExtensionParam) error {
	if len(appTplList) == 0 {
		return nil
	}
	errInfoList := []string{}
	var workers []*DeployWorker
	ar, err := NewAppRes(cluster, nil)
	if err != nil {
		return err
	}
	workerResult := make(chan WorkerResult)
	var wg sync.WaitGroup
	for _, tpl := range appTplList {
		wg.Add(1)
		wk := NewDeployWorker(tpl.GetAppName(), namespace, tpl.GetAppKind(), ar, eparam, tpl)
		workers = append(workers, wk)
		param := AppParam{Name: tpl.GetAppName()}
		go func(app AppTemplate) {
			defer wg.Done()
			err := wk.Start(tname, param)
			workerResult <- WorkerResult{
				AppName:    app.GetAppName(),
				AppVersion: app.GetAppVersion(),
				AppKind:    app.GetAppKind(),
				Result:     err,
			}
		}(tpl)
	}
	go func() {
		wg.Wait()
		close(workerResult)
	}()
	for res := range workerResult {
		if res.Result != nil {
			errInfoList = append(errInfoList, res.AppName+":"+res.Result.Error())
			beego.Error(res.Result)
		} else {
			beego.Info("deploy application " + res.AppName + " successfully!")
		}
	}
	if len(errInfoList) != 0 {
		return fmt.Errorf(strings.Join(errInfoList, ";"))
	}

	return nil
}

func AppTemplateToYamlString(tpl AppTemplate, cluster, namespace, podVersion, domainSuffix string) (string, error) {
	objs, err := tpl.GenerateKubeObject(cluster, namespace, podVersion, domainSuffix)
	if err != nil && objs == nil {
		beego.Error("generate kubernetes object failed:", err)
		return "", err
	}
	ctx := []byte{}
	elems := []reflect.Value{}
	for _, obj := range objs {
		v := reflect.ValueOf(obj)
		switch v.Kind() {
		case reflect.Ptr:
			elems = append(elems, v)
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				elems = append(elems, v.Index(i))
			}
		default:
			beego.Debug("object kind:", v.Kind())
		}
	}
	for _, elem := range elems {
		yamlBytes, err := yamlencoder.Marshal(elem.Interface())
		if err != nil {
			beego.Error("yaml marshal object failed:", err)
		}
		ctx = append(ctx, yamlBytes...)
		ctx = append(ctx, []byte(YamlSeparator)...)
	}

	return strings.TrimSuffix(string(ctx), YamlSeparator), nil
}
