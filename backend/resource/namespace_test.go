package resource

import (
	"testing"

	"kubecloud/backend/service"
	_ "kubecloud/test/mock"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	InitHarborFake()
}

func TestNamespace(t *testing.T) {
	user := "admin"
	cluster := "test"
	name := "myspace"
	desc := "this is my space"
	data := NamespaceData{
		Name:        name,
		Desc:        desc,
		CPUQuota:    "10",
		MemoryQuota: "20Gi",
	}

	client, err := service.GetClientset(cluster)
	assert.Nil(t, err)

	t.Run("Validate", func(t *testing.T) {
		err := NamespaceValidate(&data)
		assert.Nil(t, err)
	})

	t.Run("ValidateName", func(t *testing.T) {
		err := NamespaceValidate(&NamespaceData{
			Name:        "-",
			Desc:        data.Desc,
			CPUQuota:    data.CPUQuota,
			MemoryQuota: data.MemoryQuota,
		})
		assert.NotNil(t, err)
	})

	t.Run("ValidateCPUQuota", func(t *testing.T) {
		err := NamespaceValidate(&NamespaceData{
			Name:        data.Name,
			Desc:        data.Desc,
			CPUQuota:    "1kb",
			MemoryQuota: data.MemoryQuota,
		})
		assert.NotNil(t, err)
	})

	t.Run("ValidateMemoryQuota", func(t *testing.T) {
		err := NamespaceValidate(&NamespaceData{
			Name:        data.Name,
			Desc:        data.Desc,
			CPUQuota:    data.CPUQuota,
			MemoryQuota: "100c",
		})
		assert.NotNil(t, err)
	})

	t.Run("Create", func(t *testing.T) {
		row, err := NamespaceCreate(user, cluster, &data)
		assert.Nil(t, err)
		assert.NotNil(t, row)

		t.Run("CheckQuota", func(t *testing.T) {
			quotaName := GenResourceQuotaName(name)
			quota, err := client.CoreV1().ResourceQuotas(name).Get(quotaName, metav1.GetOptions{})
			assert.Nil(t, err)
			assert.NotNil(t, quota)
			assert.NotEmpty(t, quota.Spec.Hard)
			keys := []v1.ResourceName{
				v1.ResourceLimitsCPU,
				v1.ResourceRequestsCPU,
				v1.ResourceLimitsMemory,
				v1.ResourceRequestsMemory,
			}
			expected := []string{
				data.CPUQuota,
				data.CPUQuota,
				data.MemoryQuota,
				data.MemoryQuota,
			}
			for i := range keys {
				quantity, ok := quota.Spec.Hard[keys[i]]
				assert.True(t, ok)
				assert.Equal(t, expected[i], quantity.String())
			}
		})

		t.Run("Exists", func(t *testing.T) {
			_, err := NamespaceCreate(user, cluster, &data)
			assert.NotNil(t, err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		desc := "my space v2"
		row, err := NamespaceUpdate(cluster, &NamespaceData{
			Name:        name,
			Desc:        desc,
			CPUQuota:    "",
			MemoryQuota: "",
		})
		assert.Nil(t, err)
		assert.Equal(t, desc, row.Desc)
		assert.Equal(t, "", row.CPUQuota)
		assert.Equal(t, "", row.MemoryQuota)

		t.Run("CheckQuota", func(t *testing.T) {
			quotaName := GenResourceQuotaName(name)
			quota, err := client.CoreV1().ResourceQuotas(name).Get(quotaName, metav1.GetOptions{})
			assert.Nil(t, err)
			assert.NotNil(t, quota)
			assert.Empty(t, quota.Spec.Hard)
		})
	})

	t.Run("CreatePod", func(t *testing.T) {
		pod, err := client.CoreV1().Pods(name).Create(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "app",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name:  "app",
						Image: "https://foo.bar/joe/alpine:1.9",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("10"),
								v1.ResourceMemory: resource.MustParse("100Mi"),
							},
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("10"),
								v1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					},
				},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, pod)

		t.Run("DeleteNamespaceInUse", func(t *testing.T) {
			err := NamespaceDelete(cluster, name)
			assert.NotNil(t, err)
		})
	})

	t.Run("DeletePod", func(t *testing.T) {
		err := client.CoreV1().Pods(name).Delete("app", nil)
		assert.Nil(t, err)
		t.Run("DeleteNamespace", func(t *testing.T) {
			err := NamespaceDelete(cluster, name)
			assert.Nil(t, err)
		})
	})
}
