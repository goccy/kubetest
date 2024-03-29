package v1

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeepCopy(t *testing.T) {
	job := &TestJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testjob",
			Namespace: "default",
		},
		Spec: TestJobSpec{
			Repos: []RepositorySpec{
				{
					Name: "repo",
					Value: Repository{
						URL:    "https://github.com/goccy/kubetest.git",
						Branch: "master",
						Token:  "token-github",
						Merge: &MergeSpec{
							Base: "master",
						},
					},
				},
			},
			Tokens: []TokenSpec{
				{
					Name: "token-github",
					Value: TokenSource{
						GitHubToken: &GitHubTokenSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "token-github",
							},
							Key: "token",
						},
					},
				},
				{
					Name: "token-githubapp",
					Value: TokenSource{
						GitHubApp: &GitHubAppTokenSource{
							Organization: "goccy",
							AppID:        12345,
							KeyFile: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "token-githubapp",
								},
								Key: "token",
							},
						},
					},
				},
			},
			PreSteps: []PreStep{
				{
					Name: "prestep1",
					Template: TestJobTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: "prestep1",
						},
						Spec: TestJobPodSpec{
							Artifacts: []ArtifactSpec{
								{
									Name: "prestep-artifact",
									Container: ArtifactContainer{
										Name: "prestep",
										Path: filepath.Join("/", "work", "artifact.tar.gz"),
									},
								},
							},
							Containers: []TestJobContainer{
								{
									Container: corev1.Container{
										Name:       "prestep",
										Image:      "alpine",
										Command:    []string{"echo"},
										Args:       []string{"prestep"},
										WorkingDir: filepath.Join("/", "work"),
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "repo-volume",
												MountPath: filepath.Join("/", "work"),
											},
											{
												Name:      "token",
												MountPath: filepath.Join("/", "etc", "github-token"),
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
								{
									Name: "token",
									TestJobVolumeSource: TestJobVolumeSource{
										Token: &TokenVolumeSource{
											Name: "token-github",
										},
									},
								},
							},
						},
					},
				},
			},
			MainStep: MainStep{
				Strategy: &Strategy{
					Key: StrategyKeySpec{
						Env: "TEST",
						Source: StrategyKeySource{
							Dynamic: &StrategyDynamicKeySource{
								Template: TestJobTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{
										Name: "list",
									},
									Spec: TestJobPodSpec{
										Containers: []TestJobContainer{
											{
												Container: corev1.Container{
													Name:    "list",
													Image:   "alpine",
													Command: []string{"sh", "-c"},
													Args:    []string{`echo "A\nB\nC\nD"`},
												},
											},
										},
									},
								},
							},
						},
					},
					Scheduler: Scheduler{
						MaxContainersPerPod: 10,
					},
				},
				Template: TestJobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: TestJobPodSpec{
						Containers: []TestJobContainer{
							{
								Container: corev1.Container{
									Name:       "test",
									Image:      "alpine",
									Command:    []string{"sh", "-c"},
									Args:       []string{"echo $TEST"},
									WorkingDir: filepath.Join("/", "work"),
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "repo-volume",
											MountPath: filepath.Join("/", "work"),
										},
										{
											Name:      "artifact",
											MountPath: filepath.Join("/", "work", "artifact.tar.gz"),
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
							{
								Name: "artifact",
								TestJobVolumeSource: TestJobVolumeSource{
									Artifact: &ArtifactVolumeSource{
										Name: "prestep-artifact",
									},
								},
							},
						},
					},
				},
			},
			ExportArtifacts: []ExportArtifact{
				{
					Name: "prestep-artifact",
					Path: filepath.Join("/", "tmp", "artifacts"),
				},
			},
			Log: LogSpec{
				ExtParam: map[string]string{
					"key": "value",
				},
			},
		},
	}
	orig, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := json.Marshal(job.DeepCopy())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(orig, copied) {
		t.Fatalf("failed to deepcopy. expected:%q but got %q", orig, copied)
	}

	list := &TestJobList{Items: []TestJob{*job}}
	_ = list.DeepCopy()

	// for improving coverage
	_ = job.Status.DeepCopy()
	_ = job.Spec.DeepCopy()
	for _, repo := range job.Spec.Repos {
		_ = repo.DeepCopy()
		_ = repo.Value.DeepCopy()
		_ = repo.Value.Merge.DeepCopy()
	}
	for _, token := range job.Spec.Tokens {
		_ = token.DeepCopy()
		_ = token.Value.DeepCopy()
		switch {
		case token.Value.GitHubApp != nil:
			_ = token.Value.GitHubApp.DeepCopy()
		case token.Value.GitHubToken != nil:
			_ = token.Value.GitHubToken.DeepCopy()
		}
	}
	for _, prestep := range job.Spec.PreSteps {
		_ = prestep.DeepCopy()
		for _, artifact := range prestep.Template.Spec.Artifacts {
			_ = artifact.DeepCopy()
			_ = artifact.Container.DeepCopy()
		}
		for _, volume := range prestep.Template.Spec.Volumes {
			_ = volume.DeepCopy()
			_ = volume.TestJobVolumeSource.DeepCopy()
			switch {
			case volume.TestJobVolumeSource.Token != nil:
				_ = volume.TestJobVolumeSource.Token.DeepCopy()
			case volume.TestJobVolumeSource.Artifact != nil:
				_ = volume.TestJobVolumeSource.Artifact.DeepCopy()
			case volume.TestJobVolumeSource.Repo != nil:
				_ = volume.TestJobVolumeSource.Repo.DeepCopy()
			}
		}
	}
	_ = job.Spec.MainStep.Strategy.DeepCopy()
	_ = job.Spec.MainStep.Strategy.Key.DeepCopy()
	_ = job.Spec.MainStep.Strategy.Key.Source.DeepCopy()
	_ = job.Spec.MainStep.Strategy.Key.Source.Dynamic.DeepCopy()
	_ = job.Spec.MainStep.Strategy.Scheduler.DeepCopy()
	_ = job.Spec.MainStep.Template.DeepCopy()
	_ = job.Spec.MainStep.Template.Spec.DeepCopy()
	for _, volume := range job.Spec.MainStep.Template.Spec.Volumes {
		_ = volume.DeepCopy()
		_ = volume.TestJobVolumeSource.DeepCopy()
		switch {
		case volume.TestJobVolumeSource.Token != nil:
			_ = volume.TestJobVolumeSource.Token.DeepCopy()
		case volume.TestJobVolumeSource.Artifact != nil:
			_ = volume.TestJobVolumeSource.Artifact.DeepCopy()
		case volume.TestJobVolumeSource.Repo != nil:
			_ = volume.TestJobVolumeSource.Repo.DeepCopy()
		}
	}
	for _, artifact := range job.Spec.ExportArtifacts {
		_ = artifact.DeepCopy()
	}
	_ = job.Spec.Log.DeepCopy()
}
