package mock

import (
	"kubecloud/backend/models"
	"kubecloud/backend/service"
)

func init() {
	service.InitMock()
	models.InitMock()
	initBaseData()
}
