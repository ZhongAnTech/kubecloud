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
	"fmt"
	"strings"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
)

// UpdateStatusFromPod modifies the task run status based on the pod and then returns true if the pod is running and
// all sidecars are ready
func UpdateStatusFromPod(taskRun *v1alpha1.TaskRun, pod *corev1.Pod, resourceLister listers.PipelineResourceLister, kubeclient kubernetes.Interface, logger *zap.SugaredLogger) bool {
	if taskRun.Status.GetCondition(apis.ConditionSucceeded) == nil || taskRun.Status.GetCondition(apis.ConditionSucceeded).Status == corev1.ConditionUnknown {
		// If the taskRunStatus doesn't exist yet, it's because we just started running
		taskRun.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  ReasonRunning,
			Message: "Not all Steps in the Task have finished executing",
		})
	}

	taskRun.Status.PodName = pod.Name

	taskRun.Status.Steps = []v1alpha1.StepState{}
	for _, s := range pod.Status.ContainerStatuses {
		if resources.IsContainerStep(s.Name) {
			taskRun.Status.Steps = append(taskRun.Status.Steps, v1alpha1.StepState{
				ContainerState: *s.State.DeepCopy(),
				Name:           resources.TrimContainerNamePrefix(s.Name),
				ContainerName:  s.Name,
				ImageID:        s.ImageID,
			})
		}
	}

	// Complete if we did not find a step that is not complete, or the pod is in a definitely complete phase
	complete := areStepsComplete(pod) || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed

	if complete {
		updateCompletedTaskRun(taskRun, pod)
	} else {
		updateIncompleteTaskRun(taskRun, pod)
	}

	sidecarsCount, readyOrTerminatedSidecarsCount := countSidecars(pod)
	return pod.Status.Phase == corev1.PodRunning && readyOrTerminatedSidecarsCount == sidecarsCount
}

func updateCompletedTaskRun(taskRun *v1alpha1.TaskRun, pod *corev1.Pod) {
	if didTaskRunFail(pod) {
		msg := getFailureMessage(pod)
		taskRun.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  ReasonFailed,
			Message: msg,
		})
	} else {
		taskRun.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionTrue,
			Reason:  ReasonSucceeded,
			Message: "All Steps have completed executing",
		})
	}
	// update tr completed time
	taskRun.Status.CompletionTime = &metav1.Time{Time: time.Now()}
}

func updateIncompleteTaskRun(taskRun *v1alpha1.TaskRun, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		taskRun.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  ReasonRunning,
			Message: "Not all Steps in the Task have finished executing",
		})
	case corev1.PodPending:
		var reason, msg string
		if IsPodExceedingNodeResources(pod) {
			reason = ReasonExceededNodeResources
			msg = GetExceededResourcesMessage(taskRun)
		} else {
			reason = "Pending"
			msg = GetWaitingMessage(pod)
		}
		taskRun.Status.SetCondition(&apis.Condition{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  reason,
			Message: msg,
		})
	}
}

func didTaskRunFail(pod *corev1.Pod) bool {
	f := pod.Status.Phase == corev1.PodFailed
	for _, s := range pod.Status.ContainerStatuses {
		if resources.IsContainerStep(s.Name) {
			if s.State.Terminated != nil {
				f = f || s.State.Terminated.ExitCode != 0
			}
		}
	}
	return f
}

func areStepsComplete(pod *corev1.Pod) bool {
	stepsComplete := len(pod.Status.ContainerStatuses) > 0 && pod.Status.Phase == corev1.PodRunning
	for _, s := range pod.Status.ContainerStatuses {
		if resources.IsContainerStep(s.Name) {
			if s.State.Terminated == nil {
				stepsComplete = false
			}
		}
	}
	return stepsComplete
}

func countSidecars(pod *corev1.Pod) (total int, readyOrTerminated int) {
	for _, s := range pod.Status.ContainerStatuses {
		if !resources.IsContainerStep(s.Name) {
			if s.State.Running != nil && s.Ready {
				readyOrTerminated++
			} else if s.State.Terminated != nil {
				readyOrTerminated++
			}
			total++
		}
	}
	return total, readyOrTerminated
}

func getFailureMessage(pod *corev1.Pod) string {
	// First, try to surface an error about the actual build step that failed.
	for _, status := range pod.Status.ContainerStatuses {
		term := status.State.Terminated
		if term != nil && term.ExitCode != 0 {
			return fmt.Sprintf("%q exited with code %d (image: %q); for logs run: kubectl -n %s logs %s -c %s",
				status.Name, term.ExitCode, status.ImageID,
				pod.Namespace, pod.Name, status.Name)
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}
	// Lastly fall back on a generic error message.
	return "build failed for unspecified reasons."
}

func IsPodExceedingNodeResources(pod *corev1.Pod) bool {
	for _, podStatus := range pod.Status.Conditions {
		if podStatus.Reason == corev1.PodReasonUnschedulable && strings.Contains(podStatus.Message, "Insufficient") {
			return true
		}
	}
	return false
}

func GetExceededResourcesMessage(tr *v1alpha1.TaskRun) string {
	return fmt.Sprintf("TaskRun pod %q exceeded available resources", tr.Name)
}

func GetWaitingMessage(pod *corev1.Pod) string {
	// First, try to surface reason for pending/unknown about the actual build step.
	for _, status := range pod.Status.ContainerStatuses {
		wait := status.State.Waiting
		if wait != nil && wait.Message != "" {
			return fmt.Sprintf("build step %q is pending with reason %q",
				status.Name, wait.Message)
		}
	}
	// Try to surface underlying reason by inspecting pod's recent status if condition is not true
	for i, podStatus := range pod.Status.Conditions {
		if podStatus.Status != corev1.ConditionTrue {
			return fmt.Sprintf("pod status %q:%q; message: %q",
				pod.Status.Conditions[i].Type,
				pod.Status.Conditions[i].Status,
				pod.Status.Conditions[i].Message)
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}

	// Lastly fall back on a generic pending message.
	return "Pending"
}
