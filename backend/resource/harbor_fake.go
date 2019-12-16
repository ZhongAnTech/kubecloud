package resource

import "kubecloud/backend/models"

type HarborFake struct {
}

func (h *HarborFake) GetHarborProject(projectID int) (*models.ZcloudHarborProject, error) {
	return &models.ZcloudHarborProject{}, nil
}

func (h *HarborFake) GetHarborAllProject() ([]*models.ZcloudHarborProject, error) {
	return []*models.ZcloudHarborProject{}, nil
}

func (h *HarborFake) GetHarborProjectDetail(project string) (interface{}, error) {
	return "", nil
}

func (h *HarborFake) GetHarborProjectID(project string) (int, error) {
	return 0, nil
}

func (h *HarborFake) CreateHarborProject(project string) error {
	return nil
}

func (h *HarborFake) RepositoryTagExist(repository, tag string) bool {
	return true
}

func (h *HarborFake) DeleteHarborProject(project string) error {
	return nil
}

func (h *HarborFake) SetHarborProjectPublic(projectname string, public int) error {
	return nil
}

func (h *HarborFake) GetHarborRepositories(project string) ([]*models.ZcloudHarborRepository, error) {
	return []*models.ZcloudHarborRepository{}, nil
}

func (h *HarborFake) DeleteHarborRepository(repository string) error {
	return nil
}

func (h *HarborFake) GetHarborRepositoryTags(repository string) ([]models.ZcloudRepositoryTag, error) {
	return []models.ZcloudRepositoryTag{}, nil
}

func (h *HarborFake) DeleteHarborRepositoryTag(repository, tag string) error {
	return nil
}

func (h *HarborFake) HarborRepositoryConvert(repositoryData interface{}, projectData interface{}) models.ZcloudHarborRepository {
	return models.ZcloudHarborRepository{}
}

func (h *HarborFake) HarborProjectConvert(data interface{}) models.ZcloudHarborProject {
	return models.ZcloudHarborProject{}
}

func InitHarborFake() {
	harborClientProvider = func(harbor *models.ZcloudHarbor) HarborInterface {
		return &HarborFake{}
	}
}
