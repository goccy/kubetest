//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goccy/kubejob"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type PreInitCallback func(context.Context, JobExecutor) error

type Job interface {
	PreInit(corev1.Container, PreInitCallback)
	RunWithExecutionHandler(context.Context, func([]JobExecutor) error) error
	MountRepository(func(ctx context.Context, exec JobExecutor, isInitContainer bool) error)
	MountToken(func(ctx context.Context, exec JobExecutor, isInitContainer bool) error)
	MountArtifact(func(ctx context.Context, exec JobExecutor, isInitContainer bool) error)
}

type JobExecutor interface {
	Output(context.Context) ([]byte, error)
	ExecAsync(context.Context)
	Stop(context.Context) error
	CopyFrom(context.Context, string, string) error
	CopyTo(context.Context, string, string) error
	Container() corev1.Container
	Pod() *corev1.Pod
	PrepareCommand([]string) ([]byte, error)
}

type JobBuilder struct {
	cfg       *rest.Config
	namespace string
	runMode   RunMode
}

func NewJobBuilder(cfg *rest.Config, namespace string, runMode RunMode) *JobBuilder {
	return &JobBuilder{
		cfg:       cfg,
		namespace: namespace,
		runMode:   runMode,
	}
}

func (b *JobBuilder) BuildWithJob(jobSpec *batchv1.Job) (Job, error) {
	switch b.runMode {
	case RunModeKubernetes:
		job, err := kubejob.NewJobBuilder(b.cfg, b.namespace).BuildWithJob(jobSpec)
		if err != nil {
			return nil, err
		}
		return &kubernetesJob{job: job}, nil
	case RunModeLocal:
		rootDir, err := os.MkdirTemp("", "root")
		if err != nil {
			return nil, fmt.Errorf("kubetest: failed to create working directory for running on local file system")
		}
		return &localJob{rootDir: rootDir, job: jobSpec}, nil
	case RunModeDryRun:
		return &dryRunJob{job: jobSpec}, nil
	}
	return nil, fmt.Errorf("kubetest: unknown run mode %v", b.runMode)
}

type kubernetesJob struct {
	preInitCallbackContext context.Context
	job                    *kubejob.Job
	mountRepoCallback      func(context.Context, JobExecutor, bool) error
	mountTokenCallback     func(context.Context, JobExecutor, bool) error
	mountArtifactCallback  func(context.Context, JobExecutor, bool) error
}

func (j *kubernetesJob) PreInit(c corev1.Container, cb PreInitCallback) {
	j.job.PreInit(c, func(exec *kubejob.JobExecutor) error {
		return cb(j.preInitCallbackContext, &kubernetesJobExecutor{exec: exec})
	})
}

func (j *kubernetesJob) MountRepository(cb func(context.Context, JobExecutor, bool) error) {
	j.mountRepoCallback = cb
}

func (j *kubernetesJob) MountToken(cb func(context.Context, JobExecutor, bool) error) {
	j.mountTokenCallback = cb
}

func (j *kubernetesJob) MountArtifact(cb func(context.Context, JobExecutor, bool) error) {
	j.mountArtifactCallback = cb
}

func (j *kubernetesJob) SetInitContainerHook() {
	j.job.SetInitContainerExecutionHandler(func(exec *kubejob.JobExecutor) error {
		_, err := exec.ExecOnly()
		return err
	})
}

func (j *kubernetesJob) RunWithExecutionHandler(ctx context.Context, handler func([]JobExecutor) error) error {
	j.preInitCallbackContext = ctx
	j.job.SetInitContainerExecutionHandler(func(exec *kubejob.JobExecutor) error {
		if j.mountRepoCallback != nil {
			j.mountRepoCallback(ctx, &kubernetesJobExecutor{exec: exec}, true)
		}
		if j.mountTokenCallback != nil {
			j.mountTokenCallback(ctx, &kubernetesJobExecutor{exec: exec}, true)
		}
		if j.mountArtifactCallback != nil {
			j.mountArtifactCallback(ctx, &kubernetesJobExecutor{exec: exec}, true)
		}
		_, err := exec.ExecOnly()
		return err
	})
	return j.job.RunWithExecutionHandler(ctx, func(execs []*kubejob.JobExecutor) error {
		converted := make([]JobExecutor, 0, len(execs))
		for _, exec := range execs {
			e := &kubernetesJobExecutor{exec: exec}
			if j.mountRepoCallback != nil {
				j.mountRepoCallback(ctx, e, false)
			}
			if j.mountTokenCallback != nil {
				j.mountTokenCallback(ctx, e, false)
			}
			if j.mountArtifactCallback != nil {
				j.mountArtifactCallback(ctx, e, false)
			}
			converted = append(converted, e)
		}
		return handler(converted)
	})
}

type kubernetesJobExecutor struct {
	exec *kubejob.JobExecutor
}

func (e *kubernetesJobExecutor) PrepareCommand(cmd []string) ([]byte, error) {
	return e.exec.ExecPrepareCommand(cmd)
}

func (e *kubernetesJobExecutor) Output(_ context.Context) ([]byte, error) {
	return e.exec.ExecOnly()
}

func (e *kubernetesJobExecutor) ExecAsync(_ context.Context) {
	e.exec.ExecAsync()
}

func (e *kubernetesJobExecutor) Stop(_ context.Context) error {
	return e.exec.Stop()
}

func (e *kubernetesJobExecutor) CopyFrom(ctx context.Context, src string, dst string) error {
	LoggerFromContext(ctx).Debug("copy from %s on container to %s on local", src, dst)
	return e.exec.CopyFromPod(src, dst)
}

func (e *kubernetesJobExecutor) CopyTo(ctx context.Context, src string, dst string) error {
	LoggerFromContext(ctx).Debug("copy from %s on local to %s on container", src, dst)
	return e.exec.CopyToPod(src, dst)
}

func (e *kubernetesJobExecutor) Container() corev1.Container {
	return e.exec.Container
}

func (e *kubernetesJobExecutor) Pod() *corev1.Pod {
	return e.exec.Pod
}

type localJob struct {
	rootDir               string
	preInitContainer      corev1.Container
	preInitCallback       PreInitCallback
	mountRepoCallback     func(context.Context, JobExecutor, bool) error
	mountTokenCallback    func(context.Context, JobExecutor, bool) error
	mountArtifactCallback func(context.Context, JobExecutor, bool) error
	job                   *batchv1.Job
}

func (j *localJob) PreInit(c corev1.Container, cb PreInitCallback) {
	j.preInitContainer = c
	j.preInitCallback = cb
}

func (j *localJob) MountRepository(cb func(context.Context, JobExecutor, bool) error) {
	j.mountRepoCallback = cb
}

func (j *localJob) MountToken(cb func(context.Context, JobExecutor, bool) error) {
	j.mountTokenCallback = cb
}

func (j *localJob) MountArtifact(cb func(context.Context, JobExecutor, bool) error) {
	j.mountArtifactCallback = cb
}

func (j *localJob) RunWithExecutionHandler(ctx context.Context, handler func([]JobExecutor) error) error {
	preInitNameToPath := map[string]string{}
	if j.preInitCallback != nil {
		j.preInitCallback(ctx, &localJobExecutor{
			rootDir:   j.rootDir,
			container: j.preInitContainer,
		})
		for _, vm := range j.preInitContainer.VolumeMounts {
			preInitNameToPath[vm.Name] = filepath.Join(j.rootDir, vm.MountPath)
		}
	}
	execs := make([]JobExecutor, 0, len(j.job.Spec.Template.Spec.Containers))
	for _, container := range j.job.Spec.Template.Spec.Containers {
		if err := os.MkdirAll(filepath.Join(j.rootDir, container.WorkingDir), 0755); err != nil {
			return err
		}
		e := &localJobExecutor{
			rootDir:   j.rootDir,
			container: container,
		}
		if j.mountRepoCallback != nil {
			j.mountRepoCallback(ctx, e, false)
		}
		if j.mountTokenCallback != nil {
			j.mountTokenCallback(ctx, e, false)
		}
		if j.mountArtifactCallback != nil {
			j.mountArtifactCallback(ctx, e, false)
		}
		execs = append(execs, e)
	}
	return handler(execs)
}

type localJobExecutor struct {
	rootDir   string
	container corev1.Container
}

func (e *localJobExecutor) cmd(cmdarr []string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if len(cmdarr) == 1 {
		cmd = exec.Command(cmdarr[0])
	} else {
		cmd = exec.Command(cmdarr[0], cmdarr[1:]...)
	}
	for _, env := range e.container.Env {
		if env.Value == "" {
			continue
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	cmd.Dir = filepath.Join(e.rootDir, e.container.WorkingDir)
	return cmd, nil
}

func (e *localJobExecutor) PrepareCommand(cmdarr []string) ([]byte, error) {
	filteredCmd := []string{}
	for _, c := range cmdarr {
		if strings.HasPrefix(c, "/") {
			filteredCmd = append(filteredCmd, e.rootDir+c)
		} else {
			filteredCmd = append(filteredCmd, c)
		}
	}
	fmt.Println(strings.Join(filteredCmd, " "))
	cmd, err := e.cmd(filteredCmd)
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

func (e *localJobExecutor) Output(_ context.Context) ([]byte, error) {
	cmdarr := append(e.container.Command, e.container.Args...)
	if len(cmdarr) == 0 {
		return nil, fmt.Errorf("kubetest: invalid command. command is empty")
	}
	cmd, err := e.cmd(cmdarr)
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

func (e *localJobExecutor) ExecAsync(_ context.Context) {
	cmdarr := append(e.container.Command, e.container.Args...)
	if len(cmdarr) == 0 {
		return
	}
	cmd, err := e.cmd(cmdarr)
	if err != nil {
		return
	}
	go func() {
		_ = cmd.Run()
	}()
}

func (e *localJobExecutor) Stop(_ context.Context) error {
	return nil
}

func (e *localJobExecutor) CopyFrom(ctx context.Context, src string, dst string) error {
	src = filepath.Join(e.rootDir, src)
	if filepath.Base(src) != filepath.Base(dst) {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	LoggerFromContext(ctx).Debug("copy from %s on local to %s on local", src, dst)
	return localCopy(src, dst)
}

func (e *localJobExecutor) CopyTo(ctx context.Context, src string, dst string) error {
	dst = filepath.Join(e.rootDir, dst)
	if filepath.Base(src) != filepath.Base(dst) {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	LoggerFromContext(ctx).Debug("copy from %s on local to %s on local", src, dst)
	return localCopy(src, dst)
}

func (e *localJobExecutor) Container() corev1.Container {
	return e.container
}

func (e *localJobExecutor) Pod() *corev1.Pod {
	return &corev1.Pod{}
}

type dryRunJob struct {
	job *batchv1.Job
}

func (j *dryRunJob) PreInit(c corev1.Container, cb PreInitCallback) {}

func (j *dryRunJob) MountRepository(_ func(context.Context, JobExecutor, bool) error) {
}

func (j *dryRunJob) MountToken(_ func(context.Context, JobExecutor, bool) error) {
}

func (j *dryRunJob) MountArtifact(_ func(context.Context, JobExecutor, bool) error) {
}

func (j *dryRunJob) RunWithExecutionHandler(ctx context.Context, handler func([]JobExecutor) error) error {
	execs := make([]JobExecutor, 0, len(j.job.Spec.Template.Spec.Containers))
	for _, container := range j.job.Spec.Template.Spec.Containers {
		execs = append(execs, &dryRunJobExecutor{
			container: container,
		})
	}
	return handler(execs)
}

type dryRunJobExecutor struct {
	container corev1.Container
}

func (e *dryRunJobExecutor) PrepareCommand(cmd []string) ([]byte, error) {
	return nil, nil
}

func (e *dryRunJobExecutor) Output(_ context.Context) ([]byte, error) {
	return []byte("( dry running .... )"), nil
}

func (e *dryRunJobExecutor) ExecAsync(_ context.Context)  {}
func (e *dryRunJobExecutor) Stop(_ context.Context) error { return nil }
func (e *dryRunJobExecutor) CopyFrom(ctx context.Context, src string, dst string) error {
	LoggerFromContext(ctx).Debug("copy from %s on container to %s on local", src, dst)
	return nil
}

func (e *dryRunJobExecutor) CopyTo(ctx context.Context, src string, dst string) error {
	LoggerFromContext(ctx).Debug("copy from %s on local to %s on container", src, dst)
	return nil
}

func (e *dryRunJobExecutor) Container() corev1.Container {
	return e.container
}

func (e *dryRunJobExecutor) Pod() *corev1.Pod {
	return &corev1.Pod{}
}