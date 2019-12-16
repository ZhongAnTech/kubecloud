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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPVCGetCopyFromContainerSpec(t *testing.T) {
	names.TestingSeed()

	pvc := v1alpha1.ArtifactPVC{
		Name: "pipelinerun-pvc",
	}
	want := []v1alpha1.Step{{Container: corev1.Container{
		Name:    "source-copy-workspace-9l9zj",
		Image:   "override-with-bash-noop:latest",
		Command: []string{"/ko-app/bash"},
		Args:    []string{"-args", "cp -r src-path/. /workspace/destination"},
	}}}

	got := pvc.GetCopyFromStorageToSteps("workspace", "src-path", "/workspace/destination")
	if d := cmp.Diff(got, want); d != "" {
		t.Errorf("Diff:\n%s", d)
	}
}

func TestPVCGetCopyToContainerSpec(t *testing.T) {
	names.TestingSeed()

	pvc := v1alpha1.ArtifactPVC{
		Name: "pipelinerun-pvc",
	}
	want := []v1alpha1.Step{{Container: corev1.Container{
		Name:         "source-mkdir-workspace-9l9zj",
		Image:        "override-with-bash-noop:latest",
		Command:      []string{"/ko-app/bash"},
		Args:         []string{"-args", "mkdir -p /workspace/destination"},
		VolumeMounts: []corev1.VolumeMount{{MountPath: "/pvc", Name: "pipelinerun-pvc"}},
	}}, {Container: corev1.Container{
		Name:         "source-copy-workspace-mz4c7",
		Image:        "override-with-bash-noop:latest",
		Command:      []string{"/ko-app/bash"},
		Args:         []string{"-args", "cp -r src-path/. /workspace/destination"},
		VolumeMounts: []corev1.VolumeMount{{MountPath: "/pvc", Name: "pipelinerun-pvc"}},
	}}}

	got := pvc.GetCopyToStorageFromSteps("workspace", "src-path", "/workspace/destination")
	if d := cmp.Diff(got, want); d != "" {
		t.Errorf("Diff:\n%s", d)
	}
}

func TestPVCGetPvcMount(t *testing.T) {
	names.TestingSeed()
	name := "pipelinerun-pvc"
	pvcDir := "/pvc"

	want := corev1.VolumeMount{
		Name:      name,
		MountPath: pvcDir,
	}
	got := v1alpha1.GetPvcMount(name)
	if d := cmp.Diff(got, want); d != "" {
		t.Errorf("Diff:\n%s", d)
	}
}

func TestPVCGetMakeStep(t *testing.T) {
	names.TestingSeed()

	want := v1alpha1.Step{Container: corev1.Container{
		Name:    "create-dir-workspace-9l9zj",
		Image:   "override-with-bash-noop:latest",
		Command: []string{"/ko-app/bash"},
		Args:    []string{"-args", "mkdir -p /workspace/destination"},
	}}
	got := v1alpha1.CreateDirStep("workspace", "/workspace/destination")
	if d := cmp.Diff(got, want); d != "" {
		t.Errorf("Diff:\n%s", d)
	}
}

func TestStorageBasePath(t *testing.T) {
	pvc := v1alpha1.ArtifactPVC{
		Name: "pipelinerun-pvc",
	}
	pipelinerun := &v1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "pipelineruntest",
		},
	}
	got := pvc.StorageBasePath(pipelinerun)
	if d := cmp.Diff(got, "/pvc"); d != "" {
		t.Errorf("Diff:\n%s", d)
	}
}
