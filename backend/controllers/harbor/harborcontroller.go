package harbor

import (
	"fmt"
	"time"

	"kubecloud/backend/resource"

	"github.com/astaxie/beego"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubecloud/backend/dao"
	"kubecloud/backend/models"
)

// harbor controller is a global controller in system
type HarborController struct {
	cluster *models.ZcloudCluster
	harbor  *models.ZcloudHarbor
	handler func() error
}

var clusterHarbor map[string]interface{}

func harborControllerIsRunning(harborAddr string) bool {
	_, ok := clusterHarbor[harborAddr]
	return ok
}

// NewHarborController creates a new HarborController.
func NewHarborController(cluster string) (*HarborController, error) {
	info, err := dao.GetCluster(cluster)
	if err != nil {
		return nil, err
	}
	harbor, err := dao.GetHarbor(info.Registry)
	if err != nil {
		return nil, err
	}
	if harborControllerIsRunning(harbor.HarborAddr) {
		return nil, fmt.Errorf("harbor controller of this harbor is running!")
	}
	if clusterHarbor == nil {
		clusterHarbor = make(map[string]interface{})
	}
	clusterHarbor[harbor.HarborAddr] = nil
	hc := &HarborController{cluster: info, harbor: harbor}

	hc.handler = hc.syncHarbor

	return hc, nil
}

// Run begins watching and syncing.
func (hc *HarborController) Run(stopCh <-chan struct{}) {
	syncTime, err := beego.AppConfig.Int64("harbor::syncTime")
	if syncTime == 0 || err != nil {
		syncTime = 60
	}
	go wait.Until(hc.worker, time.Duration(syncTime)*time.Second, stopCh)
	<-stopCh
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (hc *HarborController) worker() {
	beego.Debug("start synchronizing harbor infomation for cluster " + hc.cluster.Name + "...")
	hc.processNextWorkItem()
	beego.Debug("finish synchronizing harbor infomation for cluster " + hc.cluster.Name + "!")
}

func (hc *HarborController) processNextWorkItem() bool {
	if err := hc.handler(); err != nil {
		beego.Warn("sync harbor failed:", err)
		return false
	}
	return true
}

func (hc *HarborController) syncHarbor() error {
	startTime := time.Now()
	defer func() {
		beego.Info(fmt.Sprintf("Finished syncing harbor %s/%q (%v)", hc.harbor.HarborId, hc.harbor.HarborName, time.Now().Sub(startTime)))
	}()

	return resource.SyncHarborRepositoryData(hc.harbor)
}
