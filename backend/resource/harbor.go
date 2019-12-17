package resource

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/common"
	"kubecloud/common/utils"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type Project_1_6_0 struct {
	ProjectID         int       `json:"project_id,omitempty"`
	OwnerID           int       `json:"owner_id,omitempty"`
	Name              string    `json:"name,omitempty"`
	CreationTime      time.Time `json:"creation_time,omitempty"`
	UpdateTime        time.Time `json:"update_time,omitempty"`
	Deleted           bool      `json:"deleted,omitempty"`
	OwnerName         string    `json:"owner_name,omitempty"`
	Togglable         bool      `json:"togglable,omitempty"`
	CurrentUserRoleID int       `json:"current_user_role_id,omitempty"`
	RepoCount         int       `json:"repo_count,omitempty"`
	Metadata          struct {
		Public string `json:"public,omitempty"`
	} `json:"metadata,omitempty"`
}

type Project_1_7_0 struct {
	ProjectID         int                    `json:"project_id"`
	OwnerID           int                    `json:"owner_id"`
	Name              string                 `json:"name"`
	CreationTime      string                 `json:"creation_time"`
	UpdateTime        string                 `json:"update_time"`
	Deleted           bool                   `json:"deleted"`
	OwnerName         string                 `json:"owner_name"`
	Togglable         bool                   `json:"togglable"`
	CurrentUserRoleID int                    `json:"current_user_role_id"`
	RepoCount         int                    `json:"repo_count"`
	ChartCount        int                    `json:"chart_count"`
	Metadata          Project_Metadata_1_7_0 `json:"metadata"`
}

type Project_Metadata_1_7_0 struct {
	Public             string `json:"public"`
	EnableContentTrust string `json:"enable_content_trust"`
	PreventVul         string `json:"prevent_vul"`
	Severity           string `json:"severity"`
	AutoScan           string `json:"auto_scan"`
}

type ProjectPost_1_6_0 struct {
	ProjectName                                string `json:"project_name,omitempty"`
	Public                                     int    `json:"public,omitempty"`
	EnableContentTrust                         bool   `json:"enable_content_trust,omitempty"`
	PreventVulnerableImagesFromRunning         bool   `json:"prevent_vulnerable_images_from_running,omitempty"`
	PreventVulnerableImagesFromRunningSeverity string `json:"prevent_vulnerable_images_from_running_severity,omitempty"`
	AutomaticallyScanImagesOnPush              bool   `json:"automatically_scan_images_on_push,omitempty"`
}

type ProjectPost_1_7_0 struct {
	ProjectName string                 `json:"project_name"`
	Metadata    Project_Metadata_1_7_0 `json:"metadata"`
}

type Repository_1_6_0 struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	ProjectID    int           `json:"project_id"`
	Description  string        `json:"description"`
	PullCount    int           `json:"pull_count"`
	StarCount    int           `json:"star_count"`
	TagsCount    int           `json:"tags_count"`
	Labels       []interface{} `json:"labels"`
	CreationTime time.Time     `json:"creation_time"`
	UpdateTime   time.Time     `json:"update_time"`
}

type Tag_1_6_0 struct {
	Digest        string    `json:"digest,omitempty"`
	Name          string    `json:"name,omitempty"`
	Size          int       `json:"size,omitempty"`
	Architecture  string    `json:"architecture,omitempty"`
	Os            string    `json:"os,omitempty"`
	DockerVersion string    `json:"docker_version,omitempty"`
	Author        string    `json:"author,omitempty"`
	Created       time.Time `json:"created,omitempty"`
	Config        struct {
		Labels map[string]string `json:"labels,omitempty"`
	} `json:"config,omitempty"`
	Signature interface{}   `json:"signature,omitempty"`
	Labels    []interface{} `json:"labels,omitempty"`
}

type ManifestConfig struct {
	Created       time.Time `json:"created"`
	DockerVersion string    `json:"docker_version"`
	Config        struct {
		Labels struct {
			BuildDate  string `json:"build-date,omitempty"`
			BranchName string `json:"branch_name,omitempty"`
		} `json:"labels,omitempty"`
	} `json:"config,omitempty"`
}

type Manifest struct {
	Manifest interface{} `json:"manifest"`
	Config   string      `json:"config"`
}

type HarborSystemInfo struct {
	WithNotary                 bool   `json:"with_notary"`
	WithClair                  bool   `json:"with_clair"`
	WithAdmiral                bool   `json:"with_admiral"`
	AdmiralEndpoint            string `json:"admiral_endpoint"`
	AuthMode                   string `json:"auth_mode"`
	RegistryURL                string `json:"registry_url"`
	ProjectCreationRestriction string `json:"project_creation_restriction"`
	SelfRegistration           bool   `json:"self_registration"`
	HasCaRoot                  bool   `json:"has_ca_root"`
	HarborVersion              string `json:"harbor_version"`
	ClairVulnerabilityStatus   struct {
		OverallLastUpdate int `json:"overall_last_update"`
		Details           []struct {
			Namespace  string `json:"namespace"`
			LastUpdate int    `json:"last_update"`
		} `json:"details"`
	} `json:"clair_vulnerability_status"`
	RegistryStorageProviderName string `json:"registry_storage_provider_name"`
	ReadOnly                    bool   `json:"read_only"`
	WithChartmuseum             bool   `json:"with_chartmuseum"`
}

func PingHarbor(addr, user, password string, https bool) error {
	method := "GET"
	protocol := "http"
	if https == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%s://%v/api/users", protocol, addr)
	if _, err := SentHarborRequestWithPassword(user, password, method, urlStr, nil); err != nil {
		return fmt.Errorf("Harbor checking connection failed: %v, please check if the harbor address or the account is correct!", err.Error())
	}
	return nil
}

func HarborList(filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	return dao.GetAllHarbor(filterQuery)
}

func HarborInspect(id string) (*models.ZcloudHarbor, error) {
	harbor, err := dao.GetHarbor(id)
	if err == orm.ErrNoRows {
		return nil, common.NewNotFound().SetCause(err)
	} else if err != nil {
		return nil, common.NewInternalServerError().SetCause(err)
	}
	return harbor, nil
}

func getHarborVersion(addr, user, password string, https bool) (string, error) {
	method := "GET"
	protocol := "http"
	if https == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%s://%v/api/systeminfo", protocol, addr)
	rsp, err := SentHarborRequestWithPassword(user, password, method, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("get harbor systeminfo failed: %v", err.Error())
	}
	var systemInfo HarborSystemInfo
	if err := json.Unmarshal(rsp, &systemInfo); err != nil {
		return "", fmt.Errorf("get harbor systeminfo failed: %v", err.Error())
	}
	return systemInfo.HarborVersion, nil
}

func HarborCreate(harborReq *models.HarborReq) error {
	if dao.HarborNameExist(harborReq.Tenant, harborReq.HarborName) {
		return common.NewBadRequest().SetCause(fmt.Errorf("harbor name is existed!"))
	}
	if err := PingHarbor(harborReq.HarborAddr, harborReq.HarborUser, harborReq.HarborPassword, harborReq.HarborHTTPSMode); err != nil {
		return common.NewBadRequest().SetCause(err)
	}

	authInfo := []byte(fmt.Sprintf("%v:%v", harborReq.HarborUser, harborReq.HarborPassword))
	authBase64 := base64.StdEncoding.EncodeToString(authInfo)
	harborVersion, err := getHarborVersion(harborReq.HarborAddr, harborReq.HarborUser, harborReq.HarborPassword, harborReq.HarborHTTPSMode)
	if err != nil {
		return err
	}
	harbor := models.ZcloudHarbor{
		Tenant:          harborReq.Tenant,
		HarborId:        harborReq.HarborId,
		HarborName:      harborReq.HarborName,
		HarborAddr:      harborReq.HarborAddr,
		HarborDesc:      harborReq.HarborDesc,
		HarborUser:      harborReq.HarborUser,
		HarborPassword:  harborReq.HarborPassword,
		HarborAuth:      authBase64,
		HarborHTTPSMode: harborReq.HarborHTTPSMode,
		HarborVersion:   harborVersion,
		Addons:          models.NewAddons(),
	}
	if harbor.HarborId == "" {
		harbor.HarborId = utils.NewUUID()
	}
	if err := dao.HarborCreate(&harbor); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	return nil
}

func HarborUpdate(harborReq *models.HarborReq) error {
	if err := PingHarbor(harborReq.HarborAddr, harborReq.HarborUser, harborReq.HarborPassword, harborReq.HarborHTTPSMode); err != nil {
		return common.NewBadRequest().SetCause(err)
	}

	timeNow, _ := time.Parse("2006-01-02 15:04:05", time.Now().Local().Format("2006-01-02 15:04:05"))
	authInfo := []byte(fmt.Sprintf("%v:%v", harborReq.HarborUser, harborReq.HarborPassword))
	authBase64 := base64.StdEncoding.EncodeToString(authInfo)

	harbor, err := dao.GetHarbor(harborReq.HarborId)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	harbor.HarborDesc = harborReq.HarborDesc
	harbor.HarborAddr = harborReq.HarborAddr
	harbor.HarborUser = harborReq.HarborUser
	harbor.HarborPassword = harborReq.HarborPassword
	harbor.HarborAuth = authBase64
	harbor.HarborHTTPSMode = harborReq.HarborHTTPSMode
	harbor.HarborVersion = harborReq.HarborVersion
	harbor.Addons.UpdateAt = timeNow

	if err := dao.HarborUpdate(harbor); err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	return nil
}

func HarborDelete(harborId string) error {
	_, err := dao.GetHarbor(harborId)
	if err == orm.ErrNoRows {
		return nil
	}
	query := utils.FilterQuery{
		IsLike: false,
	}
	query.FilterKey = "registry"
	query.FilterVal = harborId
	res, err := dao.GetClusterListByFilter(&query)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	if len(res.List.([]models.ZcloudCluster)) != 0 {
		return common.NewForbidden().SetCause(fmt.Errorf("the harbor is using by some clusters, can not be deleted!"))
	}
	err = dao.HarborDelete(harborId)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	return nil
}

type RepositoryTag struct {
	ID             int64             `json:"id"`
	Harbor         string            `json:"harbor"`
	RepositoryName string            `json:"repository_name"`
	Image          string            `json:"image"`
	Tag            string            `json:"tag"`
	CodeBranch     string            `json:"code_branch"`
	Labels         map[string]string `json:"labels"`
	CreateAt       string            `json:"create_at"`
}

func HarborRepositoryTagList(harbor, repository string, query *utils.FilterQuery) (*utils.QueryResult, error) {
	repositoryTagList := []RepositoryTag{}
	hInfo, err := dao.GetHarbor(harbor)
	if err != nil {
		return nil, err
	}
	res, err := dao.GetRepositoryTagList(harbor, repository, query)
	if err != nil {
		return nil, err
	}
	tagList, _ := res.List.([]models.ZcloudRepositoryTag)
	if len(tagList) != 0 {
		// tag list comes from database
		for _, itag := range tagList {
			labels := map[string]string{}
			if itag.Labels != "" {
				if err := json.Unmarshal([]byte(itag.Labels), &labels); err != nil {
					return nil, err
				}
			}
			repositoryTag := RepositoryTag{
				ID:             itag.ID,
				Harbor:         itag.Harbor,
				RepositoryName: itag.RepositoryName,
				Tag:            itag.Tag,
				CodeBranch:     itag.CodeBranch,
				Labels:         labels,
				CreateAt:       itag.CreateAt,
				Image:          fmt.Sprintf("%s/%s:%s", hInfo.HarborAddr, itag.RepositoryName, itag.Tag),
			}
			repositoryTagList = append(repositoryTagList, repositoryTag)
		}
		res.List = repositoryTagList
		return res, nil
	}
	list := []*models.ZcloudRepositoryTag{}
	// tag list comes from harbor server
	harborInspect, err := HarborInspect(harbor)
	if err != nil {
		return nil, err
	}
	harborHandler := NewHarborClient(harborInspect)
	tags, err := harborHandler.GetHarborRepositoryTags(repository)
	if err != nil {
		return nil, err
	}

	tagNumber := len(tags)
	// get reverse order * tags
	topCount, _ := beego.AppConfig.Int("other::topImageTagNumber")
	if topCount == 0 {
		topCount = 10
	}
	count := 0
	for i := tagNumber - 1; i >= 0 && count < topCount; i-- {
		list = append(list, &tags[i])
		count++
	}
	createAt := func(t1, t2 *models.ZcloudRepositoryTag) bool {
		return (t1.CreateAt > t2.CreateAt)
	}
	if len(list) > 0 {
		models.RepositoryTagSortBy(createAt).Sort(list)
	}
	filteVal := ""
	if query != nil {
		fmt.Sprintf("%v", query.FilterVal)
	}
	for _, itag := range list {
		labels := map[string]string{}
		if itag.Labels != "" {
			if err := json.Unmarshal([]byte(itag.Labels), &labels); err != nil {
				return nil, err
			}
		}
		repositoryTag := RepositoryTag{
			ID:             itag.ID,
			Harbor:         itag.Harbor,
			RepositoryName: itag.RepositoryName,
			Tag:            itag.Tag,
			CodeBranch:     itag.CodeBranch,
			Labels:         labels,
			CreateAt:       itag.CreateAt,
			Image:          fmt.Sprintf("%s/%s:%s", hInfo.HarborAddr, itag.RepositoryName, itag.Tag),
		}
		if filteVal != "" {
			js, err := json.Marshal(repositoryTag)
			if err != nil {
				beego.Warn("marshal repositorytag data failed:", err)
			} else {
				if !strings.Contains(string(js), filteVal) {
					continue
				}
			}
		}
		repositoryTagList = append(repositoryTagList, repositoryTag)
	}
	res.List = repositoryTagList
	return res, nil
}

func HarborEnsureImageUrl(url string) *common.Error {
	// parse url
	imageUrl := utils.ParseImageUrl(url)
	if imageUrl == nil {
		return common.NewBadRequest().SetCause(fmt.Errorf(`invalid image "%s"!`, url))
	}
	// check host
	harbor, err := dao.GetHarborByAddr(imageUrl.Host)
	if err != nil {
		return common.NewBadRequest().SetCause(fmt.Errorf(`invalid image "%s", host "%s" not found!`, url, imageUrl.Host))
	}
	res, err := HarborRepositoryTagList(harbor.HarborId, imageUrl.Name, nil)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	records, _ := res.List.([]RepositoryTag)
	for _, record := range records {
		if record.Tag == imageUrl.Tag {
			return nil
		}
	}
	// check tag (TODO: performance optimize)
	// if err := PingHarbor(harbor.HarborAddr, harbor.HarborUser, harbor.HarborPassword, harbor.HarborHTTPSMode); err != nil {
	// 	return common.NewBadRequest().SetCause(err)
	// }
	// harborClient := NewHarborClient(harbor)
	// tags, err := harborClient.GetHarborRepositoryTags(imageUrl.Name)
	// if err == nil {
	// 	for _, tag := range tags {
	// 		if tag == imageUrl.Tag {
	// 			return nil
	// 		}
	// 	}
	// 	return fmt.Errorf(`Invalid image "%s", tag "%s" not found!`, url, imageUrl.Tag)
	// }
	return common.NewBadRequest().SetCause(fmt.Errorf(`invalid image "%s", tag "%s" not found!`, url, imageUrl.Tag))
}

func SentHarborRequestWithPassword(user, password, method, urlStr string, body io.Reader) ([]byte, error) {
	rep, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	rep.Header.Set("Content-Type", "application/json")
	rep.SetBasicAuth(user, password)
	resp, err := utils.HttpClient.Do(rep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	} else if resp.StatusCode == http.StatusCreated {
		return respBody, nil
	} else if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("401 Unauthorized")
	}

	return nil, err
}

func SentHarborRequest(harborURL, method, urlStr string, body io.Reader) ([]byte, error) {
	harbor, err := dao.GetHarborByAddr(harborURL)
	if err != nil {
		return nil, err
	}
	rep, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	rep.Header.Set("Content-Type", "application/json")
	rep.SetBasicAuth(harbor.HarborUser, harbor.HarborPassword)
	resp, err := utils.HttpClient.Do(rep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	} else if resp.StatusCode == http.StatusCreated {
		return respBody, nil
	}

	return nil, err
}

type HarborInterface interface {
	GetHarborProject(projectID int) (*models.ZcloudHarborProject, error)
	GetHarborAllProject() ([]*models.ZcloudHarborProject, error)
	GetHarborProjectDetail(project string) (interface{}, error)
	GetHarborProjectID(project string) (int, error)
	CreateHarborProject(project string) error
	DeleteHarborProject(project string) error
	SetHarborProjectPublic(projectname string, public int) error
	GetHarborRepositories(project string) ([]*models.ZcloudHarborRepository, error)
	DeleteHarborRepository(repository string) error
	GetHarborRepositoryTags(repository string) ([]models.ZcloudRepositoryTag, error)
	RepositoryTagExist(repository, tag string) bool
	DeleteHarborRepositoryTag(repository, tag string) error
	HarborRepositoryConvert(repositoryData interface{}, projectData interface{}) models.ZcloudHarborRepository
	HarborProjectConvert(data interface{}) models.ZcloudHarborProject
}

var (
	harborClientProvider func(harbor *models.ZcloudHarbor) HarborInterface
)

func init() {
	harborClientProvider = func(harbor *models.ZcloudHarbor) HarborInterface {
		var harborInterface HarborInterface
		if harbor.HarborVersion == "v1.6.0" || strings.HasPrefix(harbor.HarborVersion, "v1.6.") {
			harborInterface = &Harbor_1_6_0{
				Spec: harbor,
			}

		} else if harbor.HarborVersion == "v1.7.0" || strings.HasPrefix(harbor.HarborVersion, "v1.7.") {
			harborInterface = &Harbor_1_7_0{
				Harbor_1_6_0: Harbor_1_6_0{
					Spec: harbor,
				},
			}
		} else {

			// Harbor版本如果不属于v1.7.x，将会被视为v1.6.0

			beego.Warning("harbor version %s not found in harbor version list , version v1.6.0 will be used", harbor.HarborVersion)

			harborInterface = &Harbor_1_6_0{
				Spec: harbor,
			}

		}
		return harborInterface
	}
}

func NewHarborClient(harbor *models.ZcloudHarbor) HarborInterface {
	return harborClientProvider(harbor)
}

// Harbor 1.6.0
type Harbor_1_6_0 struct {
	Spec *models.ZcloudHarbor
}

func (this *Harbor_1_6_0) CreateHarborProject(project string) error {
	method := "POST"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects", protocol, this.Spec.HarborAddr)
	jsonData, err := json.Marshal(ProjectPost_1_6_0{
		ProjectName:                        project,
		Public:                             0,
		EnableContentTrust:                 true,
		PreventVulnerableImagesFromRunning: true,
		PreventVulnerableImagesFromRunningSeverity: "",
		AutomaticallyScanImagesOnPush:              true,
	})
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(jsonData)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, body); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_6_0) DeleteHarborProject(project string) error {
	projectID, err := this.GetHarborProjectID(project)
	if err != nil {
		return err
	}

	method := "DELETE"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%v://%v/api/projects/%v", protocol, this.Spec.HarborAddr, projectID)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil); err != nil {
		return err
	}
	return nil
}

func (this *Harbor_1_6_0) SetHarborProjectPublic(projectname string, public int) error {
	projectID, err := this.GetHarborProjectID(projectname)
	if err != nil {
		return err
	}

	method := "PUT"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%v://%v/api/projects/%v/publicity", protocol, this.Spec.HarborAddr, projectID)
	project, err := this.GetHarborProject(projectID)
	if err != nil {
		return err
	}
	project.ProjectPublic = public
	jsonData, err := json.Marshal(project)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(jsonData)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, body); err != nil {
		return err
	}

	zProject, err := dao.GetHarborProject(this.Spec.HarborId, projectname)
	if err != nil {
		return err
	}
	zProject.ProjectPublic = public
	if err := dao.UpdateHarborPublicProject(zProject); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_6_0) DeleteHarborRepository(repository string) error {
	method := "DELETE"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%v://%v/api/repositories/%v", protocol, this.Spec.HarborAddr, repository)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_6_0) DeleteHarborRepositoryTag(repository, tag string) error {
	method := "DELETE"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/repositories/%v/tags/%v", protocol, this.Spec.HarborAddr, repository, tag)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_6_0) GetHarborProject(projectID int) (*models.ZcloudHarborProject, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%s://%v/api/projects/%v", protocol, this.Spec.HarborAddr, projectID)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	var project Project_1_6_0
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, err
	}

	zProject := this.HarborProjectConvert(project)
	return &zProject, nil
}

func (this *Harbor_1_6_0) GetHarborAllProject() ([]*models.ZcloudHarborProject, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects", protocol, this.Spec.HarborAddr)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	projects := []Project_1_6_0{}
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return nil, err
	}
	zProjects := []*models.ZcloudHarborProject{}
	for _, iproject := range projects {
		zProject := this.HarborProjectConvert(iproject)
		zProjects = append(zProjects, &zProject)
	}
	return zProjects, nil
}

func (this *Harbor_1_6_0) GetHarborProjectID(project string) (int, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects?name=%v", protocol, this.Spec.HarborAddr, project)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return 0, err
	}

	var projects []Project_1_6_0
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return 0, err
	}

	var projectID int
	if len(projects) == 1 {
		projectID = projects[0].ProjectID
	}
	return projectID, nil
}

func (this *Harbor_1_6_0) GetHarborProjectDetail(project string) (interface{}, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects?name=%v", protocol, this.Spec.HarborAddr, project)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return 0, err
	}

	var projects []Project_1_6_0
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return 0, err
	}

	if projects != nil {
		if len(projects) == 1 {
			return projects[0], nil
		} else {
			for _, iproject := range projects {
				if iproject.Name == project {
					return iproject, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Not found project %s", project)
}

func (this *Harbor_1_6_0) GetHarborRepositories(project string) ([]*models.ZcloudHarborRepository, error) {
	projectDetail, err := this.GetHarborProjectDetail(project)
	if err != nil {
		return nil, err
	}
	projectData := projectDetail.(Project_1_6_0)

	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/repositories?project_id=%v", protocol, this.Spec.HarborAddr, projectData.ProjectID)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	var repositoriesResp []Repository_1_6_0
	if err := json.Unmarshal(respBody, &repositoriesResp); err != nil {
		return nil, err
	}

	repositories := []*models.ZcloudHarborRepository{}
	for _, irepository := range repositoriesResp {
		repository := this.HarborRepositoryConvert(irepository, projectDetail)
		repositories = append(repositories, &repository)
	}
	return repositories, nil
}

func (this *Harbor_1_6_0) GetHarborRepositoryTags(repository string) ([]models.ZcloudRepositoryTag, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/repositories/%v/tags", protocol, this.Spec.HarborAddr, repository)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	tags := []Tag_1_6_0{}
	if err := json.Unmarshal(respBody, &tags); err != nil {
		return nil, err
	}

	tagDetail := []models.ZcloudRepositoryTag{}
	for _, itag := range tags {
		tagDetail = append(tagDetail, models.ZcloudRepositoryTag{
			Tag:            itag.Name,
			Harbor:         this.Spec.HarborId,
			RepositoryName: repository,
			CodeBranch:     itag.Config.Labels["branch_name"],
			CreateAt:       itag.Created.Local().Format("2006-01-02 15:04:05"),
		})
	}
	return tagDetail, nil
}

func (this *Harbor_1_6_0) RepositoryTagExist(repository, tag string) bool {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/repositories/%v/tags/%v", protocol, this.Spec.HarborAddr, repository, tag)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil); err != nil {
		return false
	}
	return true
}

func (this *Harbor_1_6_0) HarborRepositoryConvert(repositoryData interface{}, projectData interface{}) models.ZcloudHarborRepository {
	repository := repositoryData.(Repository_1_6_0)
	project := projectData.(Project_1_6_0)
	public := 0
	if project.Metadata.Public == "true" {
		public = 1
	}
	zRepository := models.ZcloudHarborRepository{
		Harbor:         this.Spec.HarborId,
		ProjectID:      project.ProjectID,
		ProjectName:    project.Name,
		ProjectPublic:  public,
		RepositoryName: repository.Name,
		PullCount:      repository.PullCount,
		TagsCount:      repository.TagsCount,
	}
	return zRepository
}

func (this *Harbor_1_6_0) HarborProjectConvert(data interface{}) models.ZcloudHarborProject {
	project := data.(Project_1_6_0)
	public := 1
	if project.Metadata.Public == "false" {
		public = 0
	}
	zProject := models.ZcloudHarborProject{
		Harbor:        this.Spec.HarborId,
		ProjectID:     project.ProjectID,
		ProjectName:   project.Name,
		ProjectPublic: public,
		RepoCount:     project.RepoCount,
	}
	return zProject
}

// Harbor v1.7.0
type Harbor_1_7_0 struct {
	Harbor_1_6_0
}

func (this *Harbor_1_7_0) CreateHarborProject(project string) error {
	method := "POST"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects", protocol, this.Spec.HarborAddr)
	jsonData, err := json.Marshal(ProjectPost_1_7_0{
		ProjectName: project,
		Metadata: Project_Metadata_1_7_0{
			Public: "false",
		},
	})
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(jsonData)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, body); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_7_0) DeleteHarborProject(project string) error {
	projectID, err := this.GetHarborProjectID(project)
	if err != nil {
		return err
	}

	method := "DELETE"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}
	urlStr := fmt.Sprintf("%v://%v/api/projects/%v", protocol, this.Spec.HarborAddr, projectID)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil); err != nil {
		return err
	}
	return nil
}

func (this *Harbor_1_7_0) SetHarborProjectPublic(projectname string, public int) error {
	projectID, err := this.GetHarborProjectID(projectname)
	if err != nil {
		return err
	}

	method := "PUT"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	publicStr := ""
	if public == 0 {
		publicStr = "false"
	} else if public == 1 {
		publicStr = "true"
	}
	urlStr := fmt.Sprintf("%v://%v/api/projects/%v", protocol, this.Spec.HarborAddr, projectID)
	jsonData, err := json.Marshal(ProjectPost_1_7_0{
		Metadata: Project_Metadata_1_7_0{
			Public: publicStr,
		},
	})
	if err != nil {
		return err
	}
	beego.Debug(jsonData)
	body := bytes.NewBuffer(jsonData)
	if _, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, body); err != nil {
		return err
	}

	zProject, err := dao.GetHarborProject(this.Spec.HarborId, projectname)
	if err != nil {
		return err
	}
	zProject.ProjectPublic = public
	if err := dao.UpdateHarborPublicProject(zProject); err != nil {
		return err
	}

	return nil
}

func (this *Harbor_1_7_0) GetHarborProject(projectID int) (*models.ZcloudHarborProject, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%s://%v/api/projects/%v", protocol, this.Spec.HarborAddr, projectID)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	var project Project_1_7_0
	if err := json.Unmarshal(respBody, &project); err != nil {
		return nil, err
	}

	zProject := this.HarborProjectConvert(project)
	return &zProject, nil
}

func (this *Harbor_1_7_0) GetHarborAllProject() ([]*models.ZcloudHarborProject, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects", protocol, this.Spec.HarborAddr)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	projects := []Project_1_7_0{}
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return nil, err
	}
	zProjects := []*models.ZcloudHarborProject{}
	for _, iproject := range projects {
		zProject := this.HarborProjectConvert(iproject)
		zProjects = append(zProjects, &zProject)
	}
	return zProjects, nil
}

func (this *Harbor_1_7_0) GetHarborProjectID(project string) (int, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects?name=%v", protocol, this.Spec.HarborAddr, project)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return 0, err
	}

	var projects []Project_1_7_0
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return 0, err
	}

	var projectID int
	if len(projects) == 1 {
		projectID = projects[0].ProjectID
	}
	return projectID, nil
}

func (this *Harbor_1_7_0) GetHarborProjectDetail(project string) (interface{}, error) {
	method := "GET"
	protocol := "http"
	if this.Spec.HarborHTTPSMode == true {
		protocol = "https"
	}

	urlStr := fmt.Sprintf("%v://%v/api/projects?name=%v", protocol, this.Spec.HarborAddr, project)
	respBody, err := SentHarborRequestWithPassword(this.Spec.HarborUser, this.Spec.HarborPassword, method, urlStr, nil)
	if err != nil {
		return 0, err
	}

	var projects []Project_1_7_0
	if err := json.Unmarshal(respBody, &projects); err != nil {
		return 0, err
	}

	if projects != nil {
		if len(projects) == 1 {
			return projects[0], nil
		} else {
			for _, iproject := range projects {
				if iproject.Name == project {
					return iproject, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Not found project %s", project)
}

func (this *Harbor_1_7_0) HarborProjectConvert(data interface{}) models.ZcloudHarborProject {
	project := data.(Project_1_7_0)
	public := 1
	if project.Metadata.Public == "false" {
		public = 0
	}
	zProject := models.ZcloudHarborProject{
		Harbor:        this.Spec.HarborId,
		ProjectID:     project.ProjectID,
		ProjectName:   project.Name,
		ProjectPublic: public,
		RepoCount:     project.RepoCount,
	}
	return zProject
}

func SyncHarborRepositoryData(harbor *models.ZcloudHarbor) error {
	if err := PingHarbor(harbor.HarborAddr, harbor.HarborUser, harbor.HarborPassword, harbor.HarborHTTPSMode); err != nil {
		beego.Error("ping harbor failed:", err)
		return common.NewBadRequest().SetCause(err)
	}
	harborClient := NewHarborClient(harbor)
	projects, err := harborClient.GetHarborAllProject()
	if err != nil {
		beego.Error(fmt.Sprintf("Sync harbor %v data error: %v", harbor.HarborId, err.Error()))
		return err
	}

	for _, iproject := range projects {
		if iproject.RepoCount == 0 {
			continue
		}
		if err := dao.InsertOrUpdateHarborProject(iproject); err != nil {
			beego.Warn("insert or update harbor project failed:", err)
			continue
		}
		repositories, err := harborClient.GetHarborRepositories(iproject.ProjectName)
		if err != nil {
			beego.Warn(fmt.Sprintf("Sync harbor %v data error: %v", harbor.HarborId, err.Error()))
			continue
		}
		for _, iRepository := range repositories {
			syncRepository(harborClient, harbor, iRepository)
		}
	}

	return nil
}

func syncRepository(harborClient HarborInterface, harbor *models.ZcloudHarbor, repository *models.ZcloudHarborRepository) {
	if err := dao.InsertOrUpdateRepository(repository); err != nil {
		beego.Warn(fmt.Sprintf("Sync harbor %v data error: %v", harbor.HarborId, err.Error()))
		return
	}
	tags, err := harborClient.GetHarborRepositoryTags(repository.RepositoryName)
	if err != nil {
		beego.Warn(fmt.Sprintf("Sync harbor %v data error: %v", harbor.HarborId, err.Error()))
		return
	}
	if len(tags) == 0 {
		beego.Warn("tags is empty:", harbor.HarborId, repository.RepositoryName)
		return
	}
	syncRepositoryTags(repository, tags)
}

func syncRepositoryTags(repository *models.ZcloudHarborRepository, tags []models.ZcloudRepositoryTag) {
	for _, tag := range tags {
		if err := dao.InsertOrUpdateRepositoryTag(&tag); err != nil {
			beego.Warn(fmt.Sprintf("Sync harbor %v repository %v tag failed: %v", repository.Harbor, repository.RepositoryName, err.Error()))
		}
	}
	res, err := dao.GetRepositoryTagList(repository.Harbor, repository.RepositoryName, nil)
	if err != nil {
		beego.Warn("get repository tag failed:", err)
	}
	if res.List == nil {
		beego.Warn("get repository tag empty!")
		return
	}
	repoTagList, _ := res.List.([]RepositoryTag)
	for _, repoTag := range repoTagList {
		found := false
		for _, tag := range tags {
			if repoTag.Tag == tag.Tag {
				found = true
				break
			}
		}
		if !found {
			// delete from db
			dao.DeleteRepositoryTag(repository.Harbor, repository.RepositoryName, repoTag.Tag)
		}
	}
}
