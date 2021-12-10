package v1

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func staticSources(num int) []string {
	sources := make([]string, 0, num)
	for i := 0; i < num; i++ {
		sources = append(sources, strings.Repeat("A", i))
	}
	return sources
}

func TestScheduler(t *testing.T) {
	baseTestJob := TestJob{
		ObjectMeta: testjobObjectMeta(),
		Spec: TestJobSpec{
			MainStep: MainStep{
				Strategy: &Strategy{
					Key: StrategyKeySpec{
						Env: "TEST",
					},
					Scheduler: Scheduler{
						MaxContainersPerPod:    16,
						MaxConcurrentNumPerPod: 1,
					},
				},
				Template: TestJobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-",
					},
					Spec: TestJobPodSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:       "test",
									Image:      "alpine",
									Command:    []string{"sh", "-c"},
									Args:       []string{"echo $TEST"},
									WorkingDir: filepath.Join("/", "work"),
								},
							},
						},
					},
				},
			},
		},
	}
	ctx := WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelDebug))
	t.Run("Schedule", func(t *testing.T) {
		for _, runMode := range getRunModes() {
			t.Run(runMode.String(), func(t *testing.T) {
				staticKeyNum := 848
				baseTestJob.Spec.MainStep.Strategy.Key.Source = StrategyKeySource{
					Static: staticSources(staticKeyNum),
				}
				clientset, err := kubernetes.NewForConfig(getConfig())
				if err != nil {
					t.Fatal(err)
				}
				resourceMgr := NewResourceManager(clientset, baseTestJob)
				builder := NewTaskBuilder(getConfig(), resourceMgr, "default", runMode)
				scheduler := NewTaskScheduler(baseTestJob.Spec.MainStep)
				if _, err := scheduler.Schedule(ctx, builder); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
}
