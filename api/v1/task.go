//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/kubejob"
	"github.com/lestrrat-go/backoff"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
)

type Task struct {
	Name              string
	OnFinishSubTask   func(*SubTask)
	job               Job
	copyArtifact      func(context.Context, *SubTask) error
	strategyKey       *StrategyKey
	mainContainerName string
	createJob         func(context.Context) (Job, error)
}

func (t *Task) SubTaskNum() int {
	subTaskNum := 0
	for _, c := range t.job.Spec().Template.Spec.Containers {
		if t.isMainContainer(c) {
			subTaskNum++
		}
	}
	return subTaskNum
}

func (t *Task) Run(ctx context.Context) (*TaskResult, error) {
	return t.runWithRetry(ctx)
}

func (t *Task) retryableError(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case *kubejob.PreInitError:
		return true
	case *kubejob.PendingPhaseTimeoutError:
		return true
	}
	return false
}

func (t *Task) runWithRetry(ctx context.Context) (*TaskResult, error) {
	const taskRetryCount = 2

	policy := backoff.NewExponential(
		backoff.WithInterval(1*time.Second),
		backoff.WithMaxRetries(taskRetryCount),
	)
	b, cancel := policy.Start(context.Background())
	defer cancel()

	var (
		result     *TaskResult
		err        error
		retryCount int
	)
	for backoff.Continue(b) {
		result, err = t.run(ctx)
		if err != nil {
			if t.retryableError(err) {
				LoggerFromContext(ctx).Warn(
					"failed to run task because %s. retry %d/%d",
					err, retryCount, taskRetryCount,
				)
				// Recreate the job because the internal state of the job has already changed.
				job, err := t.createJob(ctx)
				if err != nil {
					return nil, err
				}
				t.job = job
				retryCount++
				continue
			}
		}
		break
	}
	return result, err
}

func (t *Task) run(ctx context.Context) (*TaskResult, error) {
	var result TaskResult
	if err := t.job.RunWithExecutionHandler(ctx, func(executors []JobExecutor) error {
		for _, sidecar := range t.sideCarExecutors(executors) {
			sidecar.ExecAsync(ctx)
		}
		subTasks := t.getSubTasks(t.mainExecutors(executors))
		if t.strategyKey == nil {
			group, err := NewSubTaskGroup(subTasks).Run(ctx)
			if err != nil {
				return err
			}
			result.add(group)
			return nil
		}
		fmt.Println("subTasks num = ", len(subTasks))
		for _, subTaskGroup := range t.strategyKey.SubTaskScheduler.Schedule(subTasks) {
			group, err := subTaskGroup.Run(ctx)
			if err != nil {
				return err
			}
			result.add(group)
		}
		return nil
	}); err != nil {
		var failedJob *kubejob.FailedJob
		if !errors.As(err, &failedJob) {
			return nil, err
		}
		result.Err = err
	}
	return &result, nil
}

func (t *Task) getSubTasks(execs []JobExecutor) []*SubTask {
	tasks := make([]*SubTask, 0, len(execs))
	for _, exec := range execs {
		container := exec.Container()
		var envName string
		if t.strategyKey != nil {
			envName = t.strategyKey.Env
		}
		tasks = append(tasks, &SubTask{
			Name:         t.getKeyName(container),
			TaskName:     t.Name,
			KeyEnvName:   envName,
			OnFinish:     t.OnFinishSubTask,
			exec:         exec,
			copyArtifact: t.copyArtifact,
			isMain:       t.isMainExecutor(exec),
		})
	}
	return tasks
}

func (t *Task) mainExecutors(executors []JobExecutor) []JobExecutor {
	mainExecs := make([]JobExecutor, 0, len(executors))
	for _, exec := range executors {
		if t.isMainExecutor(exec) {
			mainExecs = append(mainExecs, exec)
		}
	}
	return mainExecs
}

func (t *Task) sideCarExecutors(executors []JobExecutor) []JobExecutor {
	sideCarExecs := make([]JobExecutor, 0, len(executors))
	for _, exec := range executors {
		if !t.isMainExecutor(exec) {
			sideCarExecs = append(sideCarExecs, exec)
		}
	}
	return sideCarExecs
}

func (t *Task) isMainExecutor(exec JobExecutor) bool {
	return t.isMainContainer(exec.Container())
}

func (t *Task) isMainContainer(c corev1.Container) bool {
	return t.mainContainerName == c.Name || t.hasKeyEnv(c)
}

func (t *Task) getKeyName(container corev1.Container) string {
	if t.strategyKey == nil {
		return container.Name
	}
	envName := t.strategyKey.Env
	for _, env := range container.Env {
		if env.Name == envName {
			return env.Value
		}
	}
	return container.Name
}

func (t *Task) hasKeyEnv(container corev1.Container) bool {
	if t.strategyKey == nil {
		return false
	}
	envName := t.strategyKey.Env
	for _, env := range container.Env {
		if env.Name == envName {
			return true
		}
	}
	return false
}

type TaskGroup struct {
	tasks []*Task
}

func NewTaskGroup(tasks []*Task) *TaskGroup {
	return &TaskGroup{
		tasks: tasks,
	}
}

func (g *TaskGroup) Run(ctx context.Context) (*TaskResultGroup, error) {
	var (
		eg errgroup.Group
		rg TaskResultGroup
	)
	totalSubTaskNum := 0
	for _, task := range g.tasks {
		totalSubTaskNum += task.SubTaskNum()
	}
	rg.totalSubTaskNum = totalSubTaskNum
	for _, task := range g.tasks {
		task := task
		eg.Go(func() error {
			result, err := task.Run(ctx)
			if err != nil {
				return err
			}
			rg.add(result)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return &rg, nil
}

type TaskResult struct {
	Err    error
	groups []*SubTaskResultGroup
}

func (r *TaskResult) MainTaskResults() []*SubTaskResult {
	mainResults := []*SubTaskResult{}
	for _, group := range r.groups {
		for _, result := range group.results {
			if result.IsMain {
				mainResults = append(mainResults, result)
			}
		}
	}
	return mainResults
}

func (r *TaskResult) add(group *SubTaskResultGroup) {
	r.groups = append(r.groups, group)
}

type TaskResultGroup struct {
	totalSubTaskNum int
	results         []*TaskResult
	mu              sync.Mutex
}

func (g *TaskResultGroup) TotalNum() int {
	return g.totalSubTaskNum
}

func (g *TaskResultGroup) SuccessNum() int {
	successNum := 0
	for _, result := range g.results {
		for _, group := range result.groups {
			for _, subTaskResult := range group.results {
				if subTaskResult.Status == TaskResultSuccess {
					successNum++
				}
			}
		}
	}
	return successNum
}

func (g *TaskResultGroup) FailureNum() int {
	failureNum := 0
	for _, result := range g.results {
		for _, group := range result.groups {
			for _, subTaskResult := range group.results {
				if subTaskResult.Status == TaskResultFailure {
					failureNum++
				}
			}
		}
	}
	return failureNum
}

func (g *TaskResultGroup) Status() ResultStatus {
	for _, result := range g.results {
		for _, group := range result.groups {
			for _, subTaskResult := range group.results {
				if err := subTaskResult.Error(); err != nil {
					return ResultStatusFailure
				}
			}
		}
	}
	return ResultStatusSuccess
}

func (g *TaskResultGroup) ToReportDetails() []*ReportDetail {
	details := make([]*ReportDetail, 0, g.TotalNum())
	for _, result := range g.results {
		for _, group := range result.groups {
			for _, subTaskResult := range group.results {
				details = append(details, &ReportDetail{
					Status:         subTaskResult.Status.ToResultStatus(),
					Name:           subTaskResult.Name,
					ElapsedTimeSec: int64(subTaskResult.ElapsedTime.Seconds()),
				})
			}
		}
	}
	return details
}

func (g *TaskResultGroup) add(result *TaskResult) {
	g.mu.Lock()
	g.results = append(g.results, result)
	g.mu.Unlock()
}
