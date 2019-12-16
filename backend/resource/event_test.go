package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "kubecloud/test/mock"
)

func TestEvent(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		cluster := "test"
		namespace := "tech"
		_, err := GetEvents(cluster, namespace, "", "", "", "", 10)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
	})
}
