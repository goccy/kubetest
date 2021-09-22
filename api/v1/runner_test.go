package v1

import (
	"context"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunner(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		runner := NewRunner(getConfig(), true)
		if _, err := runner.Run(context.Background(), TestJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testjob",
				Namespace: "default",
			},
			Spec: TestJobSpec{
				Repos: []RepositorySpec{
					{
						Name: "repo",
						Value: Repository{
							URL: "https://github.com/goccy/kubetest.git",
						},
					},
				},
				Template: TestJobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: TestJobPodSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:       "test",
									Image:      "alpine",
									Command:    []string{"echo"},
									Args:       []string{"hello"},
									WorkingDir: filepath.Join("/", "work"),
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "repo-volume",
											MountPath: filepath.Join("/", "work"),
										},
									},
								},
							},
						},
						Volumes: []TestJobVolume{
							{
								Name: "repo-volume",
								TestJobVolumeSource: TestJobVolumeSource{
									Repo: &RepositoryVolumeSource{
										Name: "repo",
									},
								},
							},
						},
					},
				},
			},
		}); err != nil {
			t.Fatal(err)
		}
	})
}
