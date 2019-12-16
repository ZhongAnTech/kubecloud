package models

import (
	"fmt"
	"sort"

	validate "kubecloud/common/validate"
)

type ZcloudHarbor struct {
	ID              int64  `orm:"pk;column(id);auto" json:"id"`
	HarborId        string `orm:"column(harbor_id)" json:"harbor_id"`
	Tenant          string `orm:"column(tenant)" json:"-"`
	HarborName      string `orm:"column(harbor_name);size(100)" json:"harbor_name"`
	HarborAddr      string `orm:"column(harbor_addr);size(255)" json:"harbor_addr"`
	HarborDesc      string `orm:"column(harbor_desc);size(255)" json:"harbor_desc"`
	HarborUser      string `orm:"column(harbor_user);size(255)" json:"harbor_user"`
	HarborPassword  string `orm:"column(harbor_password);size(255)" json:"harbor_password"`
	HarborAuth      string `orm:"column(harbor_auth);size(255)" json:"-"`
	HarborHTTPSMode bool   `orm:"column(harbor_https_mode);size(255)" json:"harbor_https_mode"`
	HarborVersion   string `orm:"column(harbor_version);size(255)" json:"harbor_version"`
	Addons
}

func (t *ZcloudHarbor) TableName() string {
	return "zcloud_harbor"
}

type HarborReq struct {
	Tenant          string `json:"tenant"`
	HarborId        string `json:"harbor_id"`
	HarborName      string `json:"harbor_name"`
	HarborDesc      string `json:"harbor_desc"`
	HarborAddr      string `json:"harbor_addr"`
	HarborUser      string `json:"harbor_user"`
	HarborPassword  string `json:"harbor_password"`
	HarborHTTPSMode bool   `json:"harbor_https_mode"`
	HarborVersion   string `json:"harbor_version"`
}

func (hr *HarborReq) Verify(harborReq *HarborReq) error {
	if err := validate.ValidateString(harborReq.HarborName); err != nil {
		return fmt.Errorf("harbor_name erro: %v", err.Error())
	}
	if err := validate.ValidateString(harborReq.Tenant); err != nil {
		return fmt.Errorf("harbor's tenant is not right: %v", err.Error())
	}
	return nil
}

type ZcloudHarborProject struct {
	ID            int64  `orm:"pk;column(id);auto" json:"id"`
	Harbor        string `orm:"column(harbor)" json:"harbor"`
	ProjectID     int    `orm:"column(project_id)" json:"project_id"`
	ProjectName   string `orm:"column(project_name)" json:"project_name"`
	ProjectPublic int    `orm:"column(project_public)" json:"project_public"`
	RepoCount     int    `orm:"column(repo_count)" json:"repo_count"`
}

func (t *ZcloudHarborProject) TableName() string {
	return "zcloud_harbor_project"
}

func (u *ZcloudHarborProject) TableUnique() [][]string {
	return [][]string{
		[]string{"Harbor", "ProjectID", "ProjectName"},
	}
}

type ZcloudHarborRepository struct {
	ID             int64  `orm:"pk;column(id);auto" json:"id"`
	Harbor         string `orm:"column(harbor)" json:"harbor"`
	ProjectID      int    `orm:"column(project_id)" json:"project_id"`
	ProjectName    string `orm:"column(project_name)" json:"project_name"`
	ProjectPublic  int    `orm:"column(project_public)" json:"project_public"`
	RepositoryName string `orm:"column(repository_name)" json:"repository_name"`
	PullCount      int    `orm:"column(pull_count)" json:"pull_count"`
	TagsCount      int    `orm:"column(tags_count)" json:"tags_count"`
}

func (t *ZcloudHarborRepository) TableName() string {
	return "zcloud_harbor_repository"
}

func (u *ZcloudHarborRepository) TableUnique() [][]string {
	return [][]string{
		[]string{"Harbor", "ProjectID", "ProjectName", "RepositoryName"},
	}
}

type ZcloudRepositoryTag struct {
	ID             int64  `orm:"pk;column(id);auto" json:"id"`
	Harbor         string `orm:"column(harbor)" json:"harbor"`
	RepositoryName string `orm:"column(repository_name)" json:"repository_name"`
	Tag            string `orm:"column(tag)" json:"tag"`
	CodeBranch     string `orm:"column(code_branch)" json:"code_branch"`
	Labels         string `orm:"column(labels)" json:"labels"`
	CreateAt       string `orm:"column(create_at)" json:"create_at"`
}

func (tag *ZcloudRepositoryTag) TableName() string {
	return "zcloud_repository_tag"
}

func (u *ZcloudRepositoryTag) TableUnique() [][]string {
	return [][]string{
		[]string{"Harbor", "RepositoryName", "Tag"},
	}
}

type RepositoryTagSortBy func(o1, o2 *ZcloudRepositoryTag) bool

func (by RepositoryTagSortBy) Sort(list []*ZcloudRepositoryTag) {
	ts := &repositoryTagSorter{
		Objects: list,
		by:      by,
	}
	sort.Sort(ts)
}

type repositoryTagSorter struct {
	Objects []*ZcloudRepositoryTag
	by      func(o1, o2 *ZcloudRepositoryTag) bool
}

func (s *repositoryTagSorter) Len() int {
	return len(s.Objects)
}

func (s *repositoryTagSorter) Swap(i, j int) {
	s.Objects[i], s.Objects[j] = s.Objects[j], s.Objects[i]
}

func (s *repositoryTagSorter) Less(i, j int) bool {
	return s.by(s.Objects[i], s.Objects[j])
}
