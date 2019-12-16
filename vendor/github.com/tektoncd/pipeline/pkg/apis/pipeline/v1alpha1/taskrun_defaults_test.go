/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestTaskRunSpec_SetDefaults(t *testing.T) {
	cases := []struct {
		desc string
		trs  *v1alpha1.TaskRunSpec
		want *v1alpha1.TaskRunSpec
	}{
		{
			desc: "taskref is nil",
			trs: &v1alpha1.TaskRunSpec{
				TaskRef: nil,
				Timeout: &metav1.Duration{Duration: 500 * time.Millisecond},
			},
			want: &v1alpha1.TaskRunSpec{
				TaskRef: nil,
				Timeout: &metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
		{
			desc: "taskref kind is empty",
			trs: &v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{},
				Timeout: &metav1.Duration{Duration: 500 * time.Millisecond},
			},
			want: &v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Kind: v1alpha1.NamespacedTaskKind},
				Timeout: &metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
		{
			desc: "timeout is nil",
			trs: &v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Kind: v1alpha1.ClusterTaskKind},
			},
			want: &v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Kind: v1alpha1.ClusterTaskKind},
				Timeout: &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			tc.trs.SetDefaults(ctx)

			if diff := cmp.Diff(tc.want, tc.trs); diff != "" {
				t.Errorf("Mismatch of TaskRunSpec: %s", diff)
			}
		})
	}
}

func TestTaskRunDefaulting(t *testing.T) {
	tests := []struct {
		name string
		in   *v1alpha1.TaskRun
		want *v1alpha1.TaskRun
		wc   func(context.Context) context.Context
	}{{
		name: "empty no context",
		in:   &v1alpha1.TaskRun{},
		want: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				Timeout: &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
		},
	}, {
		name: "TaskRef default to namespace kind",
		in: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo"},
			},
		},
		want: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo", Kind: v1alpha1.NamespacedTaskKind},
				Timeout: &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
		},
	}, {
		name: "TaskRef upgrade context",
		in: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo"},
			},
		},
		want: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo", Kind: v1alpha1.NamespacedTaskKind},
				Timeout: &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
		},
		wc: v1alpha1.WithUpgradeViaDefaulting,
	}, {
		name: "TaskRef default config context",
		in: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo"},
			},
		},
		want: &v1alpha1.TaskRun{
			Spec: v1alpha1.TaskRunSpec{
				TaskRef: &v1alpha1.TaskRef{Name: "foo", Kind: v1alpha1.NamespacedTaskKind},
				Timeout: &metav1.Duration{Duration: 5 * time.Minute},
			},
		},
		wc: func(ctx context.Context) context.Context {
			s := config.NewStore(logtesting.TestLogger(t))
			s.OnConfigChanged(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: config.DefaultsConfigName,
				},
				Data: map[string]string{
					"default-timeout-minutes": "5",
				},
			})
			return s.ToContext(ctx)
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			ctx := context.Background()
			if tc.wc != nil {
				ctx = tc.wc(ctx)
			}
			got.SetDefaults(ctx)
			if !cmp.Equal(got, tc.want, ignoreUnexportedResources) {
				t.Errorf("SetDefaults (-want, +got) = %v",
					cmp.Diff(got, tc.want, ignoreUnexportedResources))
			}
		})
	}
}
