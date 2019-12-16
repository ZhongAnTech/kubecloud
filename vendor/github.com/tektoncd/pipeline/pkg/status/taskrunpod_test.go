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

package status

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	fakeclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	tb "github.com/tektoncd/pipeline/test/builder"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var ignoreVolatileTime = cmp.Comparer(func(_, _ apis.VolatileTime) bool { return true })

func TestUpdateStatusFromPod(t *testing.T) {
	conditionRunning := apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  ReasonRunning,
		Message: "Not all Steps in the Task have finished executing",
	}
	conditionTrue := apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  ReasonSucceeded,
		Message: "All Steps have completed executing",
	}
	for _, c := range []struct {
		desc      string
		podStatus corev1.PodStatus
		want      v1alpha1.TaskRunStatus
	}{{
		desc:      "empty",
		podStatus: corev1.PodStatus{},

		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			Steps: []v1alpha1.StepState{},
		},
	}, {
		desc: "ignore-creds-init",
		podStatus: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init; ignored
				ImageID: "ignored",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-state-name",
				ImageID: "",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					}},
				Name:          "state-name",
				ContainerName: "step-state-name",
			}},
		},
	}, {
		desc: "ignore-init-containers",
		podStatus: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init; ignored.
				ImageID: "ignoreme",
			}, {
				// git-init; ignored.
				ImageID: "ignoreme",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-state-name",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					}},
				Name:          "state-name",
				ContainerName: "step-state-name",
				ImageID:       "image-id",
			}},
		},
	}, {
		desc: "success",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-step-push",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 0,
					},
				},
				ImageID: "image-id",
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionTrue},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 0,
					}},
				Name:          "step-push",
				ContainerName: "step-step-push",
				ImageID:       "image-id",
			}},
			// We don't actually care about the time, just that it's not nil
			CompletionTime: &metav1.Time{Time: time.Now()},
		},
	}, {
		desc: "running",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-running-step",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				Name:          "running-step",
				ContainerName: "step-running-step",
			}},
		},
	}, {
		desc: "failure-terminated",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodFailed,
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init status; ignored
				ImageID: "ignore-me",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-failure",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  ReasonFailed,
					Message: `"step-failure" exited with code 123 (image: "image-id"); for logs run: kubectl -n foo logs pod -c step-failure`,
				}},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					}},

				Name:          "failure",
				ContainerName: "step-failure",
				ImageID:       "image-id",
			}},
			// We don't actually care about the time, just that it's not nil
			CompletionTime: &metav1.Time{Time: time.Now()},
		},
	}, {
		desc: "failure-message",
		podStatus: corev1.PodStatus{
			Phase:   corev1.PodFailed,
			Message: "boom",
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  ReasonFailed,
					Message: "boom",
				}},
			},
			Steps: []v1alpha1.StepState{},
			// We don't actually care about the time, just that it's not nil
			CompletionTime: &metav1.Time{Time: time.Now()},
		},
	}, {
		desc:      "failure-unspecified",
		podStatus: corev1.PodStatus{Phase: corev1.PodFailed},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  ReasonFailed,
					Message: "build failed for unspecified reasons.",
				}},
			},
			Steps: []v1alpha1.StepState{},
			// We don't actually care about the time, just that it's not nil
			CompletionTime: &metav1.Time{Time: time.Now()},
		},
	}, {
		desc: "pending-waiting-message",
		podStatus: corev1.PodStatus{
			Phase:                 corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init status; ignored
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-status-name",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Message: "i'm pending",
					},
				},
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: `build step "step-status-name" is pending with reason "i'm pending"`,
				}},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Message: "i'm pending",
					},
				},
				Name:          "status-name",
				ContainerName: "step-status-name",
			}},
		},
	}, {
		desc: "pending-pod-condition",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Status:  corev1.ConditionUnknown,
				Type:    "the type",
				Message: "the message",
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: `pod status "the type":"Unknown"; message: "the message"`,
				}},
			},
			Steps: []v1alpha1.StepState{},
		},
	}, {
		desc: "pending-message",
		podStatus: corev1.PodStatus{
			Phase:   corev1.PodPending,
			Message: "pod status message",
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: "pod status message",
				}},
			},
			Steps: []v1alpha1.StepState{},
		},
	}, {
		desc:      "pending-no-message",
		podStatus: corev1.PodStatus{Phase: corev1.PodPending},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: "Pending",
				}},
			},
			Steps: []v1alpha1.StepState{},
		},
	}, {
		desc: "pending-not-enough-node-resources",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Reason:  corev1.PodReasonUnschedulable,
				Message: "0/1 nodes are available: 1 Insufficient cpu.",
			}},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  ReasonExceededNodeResources,
					Message: `TaskRun pod "taskRun" exceeded available resources`,
				}},
			},
			Steps: []v1alpha1.StepState{},
		},
	}, {
		desc: "with-running-sidecar",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "step-running-step",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
				{
					Name: "running-sidecar",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					Ready: true,
				},
			},
		},
		want: v1alpha1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			Steps: []v1alpha1.StepState{{
				ContainerState: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				Name:          "running-step",
				ContainerName: "step-running-step",
			}},
		},
	}} {
		t.Run(c.desc, func(t *testing.T) {
			observer, _ := observer.New(zap.InfoLevel)
			logger := zap.New(observer).Sugar()
			fakeClient := fakeclientset.NewSimpleClientset()
			sharedInfomer := informers.NewSharedInformerFactory(fakeClient, 0)
			pipelineResourceInformer := sharedInfomer.Tekton().V1alpha1().PipelineResources()
			resourceLister := pipelineResourceInformer.Lister()
			fakekubeclient := fakekubeclientset.NewSimpleClientset()

			rs := []*v1alpha1.PipelineResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "source-image",
					Namespace: "marshmallow",
				},
				Spec: v1alpha1.PipelineResourceSpec{
					Type: "image",
				},
			}}

			for _, r := range rs {
				err := pipelineResourceInformer.Informer().GetIndexer().Add(r)
				if err != nil {
					t.Errorf("pipelineResourceInformer.Informer().GetIndexer().Add(r) failed with err: %s", err)
				}
			}

			now := metav1.Now()
			p := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod",
					Namespace:         "foo",
					CreationTimestamp: now,
				},
				Status: c.podStatus,
			}
			startTime := time.Date(2010, 1, 1, 1, 1, 1, 1, time.UTC)
			tr := tb.TaskRun("taskRun", "foo", tb.TaskRunStatus(tb.TaskRunStartTime(startTime)))
			UpdateStatusFromPod(tr, p, resourceLister, fakekubeclient, logger)

			// Common traits, set for test case brevity.
			c.want.PodName = "pod"
			c.want.StartTime = &metav1.Time{Time: startTime}

			ensureTimeNotNil := cmp.Comparer(func(x, y *metav1.Time) bool {
				if x == nil {
					return y == nil
				}
				return y != nil
			})
			if d := cmp.Diff(c.want, tr.Status, ignoreVolatileTime, ensureTimeNotNil); d != "" {
				t.Errorf("Wanted:%s %v", c.desc, c.want.Conditions[0])
				t.Errorf("Diff:\n%s", d)
			}
			if tr.Status.StartTime.Time != c.want.StartTime.Time {
				t.Errorf("Expected TaskRun startTime to be unchanged but was %s", tr.Status.StartTime)
			}
		})
	}
}

func TestCountSidecars(t *testing.T) {
	tests := []struct {
		description               string
		expectedCount             int
		expectedReadyOrTerminated int
		statuses                  []corev1.ContainerStatus
	}{{
		description:               "three steps and no sidecars",
		expectedCount:             0,
		expectedReadyOrTerminated: 0,
		statuses: []corev1.ContainerStatus{
			{Name: "step-foo"},
			{Name: "step-bar"},
			{Name: "step-baz"},
		},
	}, {
		description:               "one step and two sidecars both terminated or ready",
		expectedCount:             2,
		expectedReadyOrTerminated: 2,
		statuses: []corev1.ContainerStatus{
			{Name: "step-foo"},
			{
				Name:  "sidecar-bar",
				Ready: true,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			}, {
				Name: "stopped-sidecar-baz",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 99,
					},
				},
			},
		},
	}, {
		description:               "one step and two sidecars one ready and one not",
		expectedCount:             2,
		expectedReadyOrTerminated: 1,
		statuses: []corev1.ContainerStatus{
			{Name: "step-ignore-me"},
			{
				Name:  "ready-sidecar",
				Ready: true,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			},
			{
				Name: "unready-sidecar",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			},
		},
	}}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			p := &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: test.statuses,
				},
			}
			count, readyOrTerminated := countSidecars(p)
			if count != test.expectedCount {
				t.Errorf("incorrect count of sidecars, expected %d got %d", test.expectedCount, count)
			}
			if readyOrTerminated != test.expectedReadyOrTerminated {
				t.Errorf("incorrect count of ready or terminated sidecars, expected %d got %d", test.expectedReadyOrTerminated, readyOrTerminated)
			}
		})
	}
}
