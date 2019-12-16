package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	InitMock()
}

func TestGetAppConfig(t *testing.T) {
	conf := GetAppConfig()
	env := conf.String("default::env")
	assert.Equal(t, "test", env)
}
