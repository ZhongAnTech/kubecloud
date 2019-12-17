package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestError(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		code := "oops"
		message := "don't panic"
		cause := fmt.Errorf("undefined reference")

		err := Error{}
		assert.Equal(t, 0, err.Status())

		assert.Equal(t, &err, err.SetCode(code))
		assert.Equal(t, code, err.Code())

		assert.Equal(t, &err, err.SetMessage(message))
		assert.Equal(t, message, err.Message())

		assert.Equal(t, &err, err.SetCause(cause))
		assert.Equal(t, cause, err.Cause())

		assert.NotEmpty(t, err.Error())
	})

	t.Run("4XX", func(t *testing.T) {
		errs := []*Error{
			NewBadRequest(),
			NewNotFound(),
			NewConflict(),
			NewUnauthorized(),
			NewForbidden(),
			NewMethodNotAllowed(),
			NewPayloadTooLarge(),
		}
		for _, err := range errs {
			assert.NotNil(t, err)
			assert.True(t, err.Status() >= 400 && err.Status() < 500, fmt.Sprintf(`should be 4XX: %#v`, err))
		}
	})

	t.Run("5XX", func(t *testing.T) {
		errs := []*Error{
			NewNotImplemented(),
			NewServiceUnavailable(),
			NewInternalServerError(),
		}
		for _, err := range errs {
			assert.NotNil(t, err)
			assert.True(t, err.Status() >= 500 && err.Status() < 600, fmt.Sprintf(`should be 5XX: %#v`, err))
		}
	})

	t.Run("FromRawError", func(t *testing.T) {
		rawErr := fmt.Errorf("oops")
		err := FromK8sError(rawErr)
		assert.NotNil(t, err)
		assert.Equal(t, rawErr, err.Cause())
		assert.Equal(t, NewInternalServerError().Status(), err.Status())
	})

	t.Run("FromK8sError", func(t *testing.T) {
		t.Run("NotFound", func(t *testing.T) {
			k8sErr := k8serrors.NewNotFound(v1.Resource("sc"), "not found")
			err := FromK8sError(k8sErr)
			assert.NotNil(t, err)
			assert.Equal(t, k8sErr, err.Cause())
			assert.Equal(t, NewNotFound().Status(), err.Status())
		})
	})
}
