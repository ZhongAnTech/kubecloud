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

package sidecars

import (
	"flag"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	nopImage = flag.String("nop-image", "override-with-nop:latest", "The container image used to kill sidecars")
)

type GetPod func(string, metav1.GetOptions) (*corev1.Pod, error)
type UpdatePod func(*corev1.Pod) (*corev1.Pod, error)

// Stop stops all sidecar containers inside a pod. A container is considered
// to be a sidecar if it is currently running. This func is only expected to
// be called after a TaskRun completes and all Step containers Step containers
// have already stopped.
//
// A sidecar is killed by replacing its current container image with the nop
// image, which in turn quickly exits. If the sidecar defines a command then
// it will exit with a non-zero status. When we check for TaskRun success we
// have to check for the containers we care about - not the final Pod status.
func Stop(pod *corev1.Pod, updatePod UpdatePod) error {
	updated := false
	if pod.Status.Phase == corev1.PodRunning {
		for _, s := range pod.Status.ContainerStatuses {
			if s.State.Running != nil {
				for j, c := range pod.Spec.Containers {
					if c.Name == s.Name && c.Image != *nopImage {
						updated = true
						pod.Spec.Containers[j].Image = *nopImage
					}
				}
			}
		}
	}
	if updated {
		if _, err := updatePod(pod); err != nil {
			return err
		}
	}
	return nil
}
