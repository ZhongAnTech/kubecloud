package dao

import (
	"github.com/astaxie/beego/orm"
	"kubecloud/common/utils"
	"time"

	"kubecloud/backend/models"
)

var harborEnableFilterKeys = []string{
	"harbor_name",
	"harbor_addr",
	"harbor_desc",
	"harbor_version",
	"harbor_user",
	"harbor_id",
	"harbor_https_mode",
}

var repositoryFilterKeys = []string{
	"harbor",
	"project_id",
	"project_name",
	"project_public",
	"repository_name",
	"pull_count",
	"tags_count",
}

var tagFilterKeys = []string{
	"harbor",
	"repository_name",
	"tag",
	"code_branch",
	"labels",
	"create_at",
}

func GetHarbor(id string) (*models.ZcloudHarbor, error) {
	var harbor models.ZcloudHarbor
	if err := GetOrmer().QueryTable("zcloud_harbor").
		Filter("harbor_id", id).
		Filter("deleted", 0).
		One(&harbor); err != nil {
		return nil, err
	}
	return &harbor, nil
}

func GetHarborByAddr(addr string) (*models.ZcloudHarbor, error) {
	var harbor models.ZcloudHarbor
	if err := GetOrmer().QueryTable("zcloud_harbor").
		Filter("harbor_addr", addr).
		Filter("deleted", 0).
		One(&harbor); err != nil {
		return nil, err
	}
	return &harbor, nil
}

func GetAllHarbor(filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	harbors := []models.ZcloudHarbor{}
	PageIndex := 0
	PageSize := 0
	realIndex := 0
	queryCond := orm.NewCondition().And("deleted", 0)
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(harborEnableFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
		if filterQuery.PageSize != 0 {
			if filterQuery.PageIndex > 0 {
				realIndex = filterQuery.PageIndex - 1
			}
		}
		PageIndex = filterQuery.PageIndex
		PageSize = filterQuery.PageSize
	}
	query := GetOrmer().QueryTable("zcloud_harbor").OrderBy("-update_at").SetCond(queryCond)
	if PageSize != 0 {
		query = query.Limit(PageSize, PageSize*realIndex)
	}
	if _, err := query.All(&harbors); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: PageIndex,
			PageSize:  PageSize,
		},
		List: harbors}, err
}

func HarborCreate(harbor *models.ZcloudHarbor) error {
	if _, err := GetOrmer().Insert(harbor); err != nil {
		return err
	}
	return nil
}

func HarborUpdate(harbor *models.ZcloudHarbor) error {
	if _, err := GetOrmer().Update(harbor,
		"harbor_addr", "harbor_desc",
		"harbor_user", "harbor_password",
		"harbor_auth", "update_at", "harbor_version",
		"harbor_https_mode"); err != nil {
		return err
	}
	return nil
}

func HarborDelete(harbor string) error {
	if _, err := GetOrmer().Raw("update zcloud_harbor set deleted=1,delete_at=Now() where harbor_id=? and deleted=0",
		harbor).Exec(); err != nil {
		return err
	}
	return nil
}

func HarborNameExist(tenant, harborName string) bool {
	exist := GetOrmer().QueryTable("zcloud_harbor").
		Filter("tenant", tenant).
		Filter("harbor_name", harborName).Filter("deleted", 0).Exist()
	return exist
}

func HarborIsExist(tenant, harbor string) bool {
	exist := GetOrmer().QueryTable("zcloud_harbor").Filter("tenant", tenant).Filter("harbor_id", harbor).Filter("deleted", 0).Exist()
	return exist
}

func HarborAddrExist(harborAddr string) bool {
	exist := GetOrmer().QueryTable("zcloud_harbor").Filter("harbor_addr", harborAddr).Filter("deleted", 0).Exist()
	return exist
}

func InsertOrUpdateHarborProject(project *models.ZcloudHarborProject) error {
	_, err := GetOrmer().InsertOrUpdate(project)
	return err
}

// func GetHarborPublicProject(harborName string) ([]*models.ZcloudHarborProject, error) {
// 	projects := []*models.ZcloudHarborProject{}
// 	_, err := GetOrmer().QueryTable("zcloud_harbor_project").Filter("harbor_name", harborName).Filter("project_public", 1).All(&projects)
// 	return projects, err
// }

func GetHarborProject(harbor, projectName string) (*models.ZcloudHarborProject, error) {
	var project models.ZcloudHarborProject
	err := GetOrmer().QueryTable("zcloud_harbor_project").Filter("harbor", harbor).Filter("project_name", projectName).One(&project)
	return &project, err
}

func UpdateHarborPublicProject(project *models.ZcloudHarborProject) error {
	_, err := GetOrmer().Update(project)
	if _, err := GetOrmer().Raw("update zcloud_harbor_repository set project_public=? where harbor=? and project_name=?",
		project.ProjectPublic, project.Harbor, project.ProjectName).Exec(); err != nil {
		return err
	}
	return err
}

func InsertOrUpdateRepository(repository *models.ZcloudHarborRepository) error {
	_, err := GetOrmer().InsertOrUpdate(repository)
	return err
}

func IsPublicHarborProject(harbor, projectName string) bool {
	var project models.ZcloudHarborProject
	err := GetOrmer().QueryTable("zcloud_harbor_project").Filter("harbor", harbor).Filter("project_name", projectName).One(&project)
	if err != nil || project.ProjectPublic == 0 {
		return false
	}
	return true
}

func DeleteRepository(harbor, repositoryName string) error {
	repository, err := GetRepositories(harbor, repositoryName)
	if err != nil {
		return err
	}
	if _, err := GetOrmer().Delete(repository); err != nil {
		return err
	}
	return nil
}

func GetRepositories(harbor, repositoryName string) (*models.ZcloudHarborRepository, error) {
	var repository models.ZcloudHarborRepository
	err := GetOrmer().QueryTable("zcloud_harbor_repository").
		Filter("harbor", harbor).
		Filter("repository_name", repositoryName).
		One(&repository)
	return &repository, err
}

func GetRepositoriesList(harbor, public string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	repositories := []models.ZcloudHarborRepository{}
	queryCond := orm.NewCondition().And("harbor", harbor)
	switch public {
	case "false":
		queryCond = queryCond.And("project_public", 0)
	case "true":
		queryCond = queryCond.And("project_public", 1)
	}
	PageIndex := 0
	PageSize := 0
	realIndex := 0
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(repositoryFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
		if filterQuery.PageSize != 0 {
			if filterQuery.PageIndex > 0 {
				realIndex = filterQuery.PageIndex - 1
			}
		}
		PageIndex = filterQuery.PageIndex
		PageSize = filterQuery.PageSize
	}
	query := GetOrmer().QueryTable("zcloud_harbor_repository").SetCond(queryCond)
	if PageSize != 0 {
		query = query.Limit(PageSize, PageSize*realIndex)
	}
	if _, err := query.All(&repositories); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: PageIndex,
			PageSize:  PageSize,
		},
		List: repositories}, err
}

// insert or update a image's tag
func InsertOrUpdateRepositoryTag(tag *models.ZcloudRepositoryTag) error {
	var err error
	exist := GetOrmer().QueryTable((&models.ZcloudRepositoryTag{}).TableName()).Filter("harbor", tag.Harbor).
		Filter("repository_name", tag.RepositoryName).Filter("tag", tag.Tag).Exist()
	if !exist {
		_, err = GetOrmer().Insert(tag)
	} else {
		sql := `update ` + (&models.ZcloudRepositoryTag{}).TableName() + ` set code_branch=?,create_at=? where harbor=? and repository_name=? and tag=?`
		_, err = GetOrmer().Raw(sql, tag.CodeBranch, tag.CreateAt, tag.Harbor, tag.RepositoryName, tag.Tag).Exec()
	}
	return err
}

// delete all image's tags
func DeleteRepositoryTags(harbor, repositoryName string) error {
	sql := "delete from " + (&models.ZcloudRepositoryTag{}).TableName() + " where harbor=? and repository_name=?"
	_, err := GetOrmer().Raw(sql, harbor, repositoryName).Exec()

	return err
}

// delete all image's tags
func DeleteRepositoryTag(harbor, repositoryName, tag string) error {
	sql := "delete from " + (&models.ZcloudRepositoryTag{}).TableName() + " where harbor=? and repository_name=? and tag=?"
	_, err := GetOrmer().Raw(sql, harbor, repositoryName, tag).Exec()

	return err
}

// get image's tag list
func GetRepositoryTagList(harbor, repositoryName string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	tagList := []models.ZcloudRepositoryTag{}
	queryCond := orm.NewCondition().And("harbor", harbor).And("repository_name", repositoryName)
	PageIndex := 0
	PageSize := 0
	realIndex := 0
	if filterQuery != nil {
		filterCond := filterQuery.FilterCondition(tagFilterKeys)
		if filterCond != nil {
			queryCond = queryCond.AndCond(filterCond)
		}
		if filterQuery.PageSize != 0 {
			if filterQuery.PageIndex > 0 {
				realIndex = filterQuery.PageIndex - 1
			}
		}
		PageIndex = filterQuery.PageIndex
		PageSize = filterQuery.PageSize
	}
	query := GetOrmer().QueryTable((&models.ZcloudRepositoryTag{}).TableName()).OrderBy("-create_at").SetCond(queryCond)
	if PageSize != 0 {
		query = query.Limit(PageSize, PageSize*realIndex)
	}
	if _, err := query.All(&tagList); err != nil {
		return nil, err
	}
	count, err := query.Count()
	if err != nil {
		return nil, err
	}
	return &utils.QueryResult{
		Base: utils.PageInfo{
			TotalNum:  count,
			PageIndex: PageIndex,
			PageSize:  PageSize,
		},
		List: tagList}, err
}

func GetRepositoryTagDeatil(harbor, repositoryName, tag string) (*models.ZcloudRepositoryTag, error) {
	var tagDeatil models.ZcloudRepositoryTag
	err := GetOrmer().QueryTable((&models.ZcloudRepositoryTag{}).TableName()).
		Filter("harbor", harbor).
		Filter("repository_name", repositoryName).
		Filter("tag", tag).One(&tagDeatil)
	return &tagDeatil, err
}

func SetRepositoryTagLabels(harbor, repositoryName, tag, labels string) error {
	var tagDetail *models.ZcloudRepositoryTag
	var err error
	exist := true
	tagDetail, err = GetRepositoryTagDeatil(harbor, repositoryName, tag)
	if err != nil {
		if err == orm.ErrNoRows {
			exist = false
		} else {
			return err
		}
	}
	if exist {
		tagDetail.Labels = labels
		if _, err = GetOrmer().Update(tagDetail); err != nil {
			return err
		}
	} else {
		tagDetail = &models.ZcloudRepositoryTag{
			Harbor:         harbor,
			RepositoryName: repositoryName,
			Tag:            tag,
			Labels:         labels,
			CreateAt:       time.Now().Local().Format("2006-01-02 15:04:05"),
		}
		if _, err = GetOrmer().Insert(tagDetail); err != nil {
			return err
		}
	}
	return err
}
