package v1

import (
	"context"
	"fmt"
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
	t.Run("ScheduleTask", func(t *testing.T) {
		for _, runMode := range getRunModes() {
			t.Run(runMode.String(), func(t *testing.T) {
				t.Run("less than maxContainersPerPod", func(t *testing.T) {
					staticKeyNum := 10
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
				t.Run("mod less than maxContainersPerPod", func(t *testing.T) {
					staticKeyNum := 31
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
				t.Run("mod equal maxContainersPerPod", func(t *testing.T) {
					staticKeyNum := 32
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
				t.Run("mod greater than maxContainersPerPod", func(t *testing.T) {
					staticKeyNum := 33
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
			})
		}
	})
	t.Run("ScheduleSubTask", func(t *testing.T) {
		for _, test := range []struct {
			maxConcurrentNumPerPod int
			taskNum                int
			expectedGroupNum       int
		}{
			{maxConcurrentNumPerPod: 1, taskNum: 10, expectedGroupNum: 10},
			{maxConcurrentNumPerPod: 2, taskNum: 2, expectedGroupNum: 1},
			{maxConcurrentNumPerPod: 2, taskNum: 3, expectedGroupNum: 2},
			{maxConcurrentNumPerPod: 2, taskNum: 9, expectedGroupNum: 5},
			{maxConcurrentNumPerPod: 2, taskNum: 10, expectedGroupNum: 5},
			{maxConcurrentNumPerPod: 2, taskNum: 11, expectedGroupNum: 6},
			{maxConcurrentNumPerPod: 4, taskNum: 11, expectedGroupNum: 3},
			{maxConcurrentNumPerPod: 12, taskNum: 12, expectedGroupNum: 1},
		} {
			name := fmt.Sprintf(
				"maxConcurrentNumPerPod_%d_taskNum_%d",
				test.maxConcurrentNumPerPod,
				test.taskNum,
			)
			t.Run(name, func(t *testing.T) {
				subtasks := make([]*SubTask, test.taskNum)
				groups := NewSubTaskScheduler(test.maxConcurrentNumPerPod).Schedule(subtasks)
				if len(groups) != test.expectedGroupNum {
					t.Fatalf("failed to schedule subtask. expected: %d but got %d", test.expectedGroupNum, len(groups))
				}
				sum := 0
				for _, group := range groups {
					sum += len(group.tasks)
				}
				if sum != test.taskNum {
					t.Fatalf("failed to schedul subtask: expected %d but got %d", test.taskNum, sum)
				}
			})
		}
	})
}
