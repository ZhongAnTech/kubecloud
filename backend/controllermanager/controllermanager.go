package controllermanager

import (
	"context"
	"github.com/astaxie/beego"
	"github.com/golang/glog"

	"math/rand"
	"os"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	componentbaseconfig "k8s.io/component-base/config"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

var ctrollerContextList map[string]ControllerContext

func Init() {
	ctrollerContextList = make(map[string]ControllerContext)
	RunDefaultClusterControllers()
}

type ControllerOption struct {
	NormalConcurrentSyncs   int
	MinInformerResyncPeriod time.Duration
	// leaderElection defines the configuration of leader election client.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
}

type ControllerContext struct {
	Cluster string
	// ClientBuilder will provide a client for this controller to use
	Client kubernetes.Interface

	// InformerFactory gives access to informers for the controller.
	InformerFactory informers.SharedInformerFactory

	// Options provides access to init options for a given controller
	Option ControllerOption

	// Stop is the stop channel
	Stop <-chan struct{}
}

func GetDefualtControllerOption() ControllerOption {
	return ControllerOption{
		NormalConcurrentSyncs:   1,
		MinInformerResyncPeriod: 12 * time.Hour,
		LeaderElection: componentbaseconfig.LeaderElectionConfiguration{
			LeaseDuration: metav1.Duration{Duration: 15 * time.Second},
			RenewDeadline: metav1.Duration{Duration: 10 * time.Second},
			RetryPeriod:   metav1.Duration{Duration: 2 * time.Second},
			ResourceLock:  componentbaseconfigv1alpha1.EndpointsResourceLock,
			LeaderElect:   true,
		},
	}
}

// CreateControllerContext creates a context struct containing references to resources needed by the
func CreateControllerContext(cluster string, option ControllerOption, stop <-chan struct{}) (*ControllerContext, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error("Start controllers failed: ", err)
		return nil, err
	}
	sharedInformers := informers.NewSharedInformerFactory(client, resyncPeriod(option.MinInformerResyncPeriod)())
	ctx := ControllerContext{
		Cluster:         cluster,
		Client:          client,
		InformerFactory: sharedInformers,
		Option:          option,
		Stop:            stop,
	}
	return &ctx, nil
}

func setControllerContextStop(ctx *ControllerContext, stop <-chan struct{}) {
	ctx.Stop = stop
}

// ResyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func resyncPeriod(period time.Duration) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(period.Nanoseconds()) * factor)
	}
}

func createRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.CoreV1().RESTClient()).Events("")})
	return eventBroadcaster.NewRecorder(scheme.Scheme, core.EventSource{Component: "zcloud"})
}

func StartControllers(cluster string) {
	if _, exist := ctrollerContextList[cluster]; exist {
		beego.Info("Controllers for " + cluster + " has started!")
		return
	}
	ctx, err := CreateControllerContext(cluster, GetDefualtControllerOption(), nil)
	if err != nil {
		beego.Error("Start controllersfor "+cluster+" failed: ", err)
		return
	}
	run := func(context context.Context) {
		setControllerContextStop(ctx, context.Done())
		ctrollerContextList[cluster] = *ctx
		controllers := GetControllerList()
		for controllerName, run := range controllers {
			beego.Info("Starting controller", controllerName, "for cluster: "+cluster)
			err := run(*ctx)
			if err != nil {
				beego.Error("Starting controller "+controllerName, "for cluster: "+cluster, "failed:", err)
			} else {
				beego.Info("Started controller", controllerName, "for cluster: "+cluster)
			}
		}
		ctx.InformerFactory.Start(context.Done())
	}
	id, err := os.Hostname()
	if err != nil {
		beego.Error(err)
		return
	}
	beego.Debug("hostname: ", id)
	namespace := "kubecloud"
	name := cluster + "-" + "kubecloud-controller-manager"
	rl, err := resourcelock.New(ctx.Option.LeaderElection.ResourceLock,
		namespace,
		name,
		ctx.Client.CoreV1(),
		ctx.Client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: createRecorder(ctx.Client),
		})
	if err != nil {
		beego.Error("error creating lock: ", err)
		return
	}
	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: ctx.Option.LeaderElection.LeaseDuration.Duration,
		RenewDeadline: ctx.Option.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:   ctx.Option.LeaderElection.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				beego.Error("leaderelection lost for cluster", cluster)
			},
		},
	})
	select {
	case <-ctx.Stop:
		delete(ctrollerContextList, cluster)
	}
	StartControllers(cluster)
}

func RunDefaultClusterControllers() {
	disable, _ := service.GetAppConfig().Bool("k8s::syncResourceDisable")
	if disable {
		return
	}
	clusters, err := dao.GetAllClusters()
	if err != nil {
		beego.Error("Cant run controllers for: ", err)
		return
	}
	for _, cluster := range clusters {
		if cluster.Status == models.ClusterStatusRunning {
			go func(id string) {
				StartControllers(id)
			}(cluster.ClusterId)
		}
	}
}
