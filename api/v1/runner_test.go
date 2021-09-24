package v1

import (
	"context"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testjobObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "testjob",
		Namespace: "default",
	}
}

func testRepos() []RepositorySpec {
	return []RepositorySpec{
		{
			Name: "repo",
			Value: Repository{
				URL: "https://github.com/goccy/kubetest.git",
			},
		},
	}
}

func testRepoVolume() TestJobVolume {
	return TestJobVolume{
		Name: "repo-volume",
		TestJobVolumeSource: TestJobVolumeSource{
			Repo: &RepositoryVolumeSource{
				Name: "repo",
			},
		},
	}
}

func testRepoVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "repo-volume",
		MountPath: filepath.Join("/", "work"),
	}
}

func TestRunner(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		runner := NewRunner(getConfig(), RunModeLocal)
		if _, err := runner.Run(context.Background(), TestJob{
			ObjectMeta: testjobObjectMeta(),
			Spec: TestJobSpec{
				Repos: testRepos(),
				Template: TestJobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: TestJobPodSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:         "test",
									Image:        "alpine",
									Command:      []string{"echo"},
									Args:         []string{"hello"},
									WorkingDir:   filepath.Join("/", "work"),
									VolumeMounts: []corev1.VolumeMount{testRepoVolumeMount()},
								},
							},
						},
						Volumes: []TestJobVolume{testRepoVolume()},
					},
				},
			},
		}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("prestep", func(t *testing.T) {
		runner := NewRunner(getConfig(), RunModeLocal)
		if _, err := runner.Run(context.Background(), TestJob{
			ObjectMeta: testjobObjectMeta(),
			Spec: TestJobSpec{
				Repos: testRepos(),
				PreSteps: []PreStep{
					{
						Name: "build",
						Template: TestJobTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Name: "build",
							},
							Spec: TestJobPodSpec{
								Artifacts: []ArtifactSpec{
									{
										Name: "build-test",
										Container: ArtifactContainer{
											Name: "build",
											Path: filepath.Join("/", "work", "v1.test"),
										},
									},
								},
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:    "build",
											Image:   "alpine",
											Command: []string{"go"},
											Args: []string{
												"test",
												"-c",
												"./api/v1",
											},
											WorkingDir:   filepath.Join("/", "work"),
											VolumeMounts: []corev1.VolumeMount{testRepoVolumeMount()},
										},
									},
								},
								Volumes: []TestJobVolume{testRepoVolume()},
							},
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
									Command:    []string{"ls"},
									Args:       []string{"compiled.test"},
									WorkingDir: filepath.Join("/", "work"),
									VolumeMounts: []corev1.VolumeMount{
										testRepoVolumeMount(),
										{
											Name:      "build-artifact",
											MountPath: filepath.Join("/", "work", "compiled.test"),
										},
									},
								},
							},
						},
						Volumes: []TestJobVolume{
							testRepoVolume(),
							{
								Name: "build-artifact",
								TestJobVolumeSource: TestJobVolumeSource{
									Artifact: &ArtifactVolumeSource{
										Name: "build-test",
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
