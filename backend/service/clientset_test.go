package service

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

func init() {
	InitMock()
}

func TestGetClientset(t *testing.T) {
	nproc := 10
	clusters := []string{"a", "b", "c", "d"}
	t.Parallel()
	for _, cluster := range clusters {
		cluster := cluster
		slice := make([]kubernetes.Interface, nproc)
		t.Run(cluster, func(t *testing.T) {
			t.Parallel()
			for i := 0; i < nproc; i++ {
				i := i
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					clientset, err := GetClientset(cluster)
					assert.Nil(t, err)
					assert.NotNil(t, clientset)
					slice[i] = clientset
				})
			}

		})
		for i := 1; i < nproc; i++ {
			if slice[0] != slice[i] {
				assert.Equal(t, slice[0], slice[i])
			}
		}
	}

}
