package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "kubecloud/test/mock"
)

func TestCongfigmap(t *testing.T) {
	cluster := "test"
	namespace := "tech"
	configMapSpec := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Data: map[string]string{
			"test.conf": "test",
		},
	}

	t.Run("Create", func(t *testing.T) {
		result, err := ConfigMapCreate(cluster, namespace, &configMapSpec)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		assert.Equal(t, configMapSpec.ObjectMeta.Name, result.ObjectMeta.Name)
	})
	t.Run("List", func(t *testing.T) {
		result, err := ConfigMapList(cluster, []string{namespace})
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		assert.NotZero(t, result)
	})
	t.Run("Inspect", func(t *testing.T) {
		result, err := ConfigMapInspect(cluster, namespace, configMapSpec.ObjectMeta.Name)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		assert.Equal(t, configMapSpec.ObjectMeta.Name, result.ObjectMeta.Name)
	})
	t.Run("Update", func(t *testing.T) {
		configMapSpec.Data["test.conf"] = "test-update"
		result, err := ConfigMapUpdate(cluster, namespace, configMapSpec.ObjectMeta.Name, &configMapSpec)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		assert.Equal(t, result.Data["test.conf"], "test-update")
	})
	t.Run("Delete", func(t *testing.T) {
		err := ConfigMapDelete(cluster, namespace, configMapSpec.ObjectMeta.Name)
		assert.Nil(t, err)
		if err != nil {
			t.Fatalf("get failed:%v", err)
		}
		t.Run("Inspect", func(t *testing.T) {
			_, err := ConfigMapInspect(cluster, namespace, configMapSpec.ObjectMeta.Name)
			assert.NotNil(t, err)
		})
	})
}
