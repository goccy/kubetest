// +build !ignore_autogenerated

package v1

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/goccy/kubejob"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	gitImageName         = "alpine/git"
	oauthTokenEnv        = "OAUTH_TOKEN"
	sharedVolumeName     = "repo"
	defaultConcurrentNum = 4
	defaultListDelimiter = "\n"
)

var (
	ErrFailedTestJob = xerrors.New("failed test job")
)

type TestJobRunner struct {
	*kubernetes.Clientset
	token                     string
	disabledPrepareLog        bool
	disabledCommandLog        bool
	logger                    func(*kubejob.ContainerLog)
	containerNameToCommandMap sync.Map
}

func NewTestJobRunner(clientset *kubernetes.Clientset) *TestJobRunner {
	return &TestJobRunner{
		Clientset: clientset,
	}
}

func (r *TestJobRunner) SetToken(token string) {
	r.token = token
}

func (r *TestJobRunner) sharedVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: sharedVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
}

func (r *TestJobRunner) sharedVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      sharedVolumeName,
		MountPath: filepath.Join("/", "git", "workspace"),
	}
}

func (r *TestJobRunner) gitImage(job TestJob) string {
	if job.Spec.GitImage != "" {
		return job.Spec.GitImage
	}
	return gitImageName
}

func (r *TestJobRunner) cloneURL(job TestJob) string {
	repo := job.Spec.Repo
	if r.token != "" {
		return fmt.Sprintf("https://$(%s)@%s.git", oauthTokenEnv, repo)
	}
	return fmt.Sprintf("https://%s.git", repo)
}

func (r *TestJobRunner) gitCloneContainer(job TestJob) apiv1.Container {
	cloneURL := r.cloneURL(job)
	cloneCmd := []string{"clone"}
	volumeMount := r.sharedVolumeMount()
	branch := job.Spec.Branch
	if branch != "" {
		cloneCmd = append(cloneCmd, "-b", branch, cloneURL, volumeMount.MountPath)
	} else {
		cloneCmd = append(cloneCmd, cloneURL, volumeMount.MountPath)
	}
	return apiv1.Container{
		Name:         "kubetest-init-clone",
		Image:        r.gitImage(job),
		Command:      []string{"git"},
		Args:         cloneCmd,
		Env:          []apiv1.EnvVar{{Name: oauthTokenEnv, Value: r.token}},
		VolumeMounts: []apiv1.VolumeMount{volumeMount},
	}
}

func (r *TestJobRunner) gitSwitchContainer(job TestJob) apiv1.Container {
	volumeMount := r.sharedVolumeMount()
	return apiv1.Container{
		Name:         "kubetest-init-switch",
		Image:        r.gitImage(job),
		WorkingDir:   volumeMount.MountPath,
		Command:      []string{"git"},
		Args:         []string{"checkout", "--detach", job.Spec.Rev},
		VolumeMounts: []apiv1.VolumeMount{volumeMount},
	}
}

func (r *TestJobRunner) initContainers(job TestJob) []apiv1.Container {
	if job.Spec.Branch != "" {
		return []apiv1.Container{r.gitCloneContainer(job)}
	}
	return []apiv1.Container{
		r.gitCloneContainer(job),
		r.gitSwitchContainer(job),
	}
}

func (r *TestJobRunner) command(cmd Command) ([]string, []string) {
	e := base64.StdEncoding.EncodeToString([]byte(string(cmd)))
	return []string{"sh"}, []string{"-c", fmt.Sprintf("echo %s | base64 -d | sh", e)}
}

func (r *TestJobRunner) commandText(cmd Command) string {
	c, args := r.command(cmd)
	return strings.Join(append(c, args...), " ")
}

func (r *TestJobRunner) DisablePrepareLog() {
	r.disabledPrepareLog = true
}

func (r *TestJobRunner) DisableCommandLog() {
	r.disabledCommandLog = true
}

func (r *TestJobRunner) SetLogger(logger func(*kubejob.ContainerLog)) {
	r.logger = logger
}

func (r *TestJobRunner) Run(ctx context.Context, testjob TestJob) error {
	if testjob.Spec.Branch == "" && testjob.Spec.Rev == "" {
		testjob.Spec.Branch = "master"
	}
	token := testjob.Spec.Token
	if token != nil {
		secret, err := r.CoreV1().
			Secrets(testjob.Namespace).
			Get(token.SecretKeyRef.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		data, exists := secret.Data[token.SecretKeyRef.Key]
		if !exists {
			gr := schema.GroupResource{
				Group:    GroupVersion.Group,
				Resource: "TestJob",
			}
			return errors.NewNotFound(gr, "token")
		}
		r.token = strings.TrimSpace(string(data))
	}
	if err := r.prepare(ctx, testjob); err != nil {
		return err
	}
	if testjob.Spec.DistributedTest != nil {
		return r.runDistributedTest(ctx, testjob)
	}
	return r.run(ctx, testjob)
}

func (r *TestJobRunner) prepareImage(stepIdx int, testjob TestJob) string {
	step := testjob.Spec.Prepare.Steps[stepIdx]
	if step.Image != "" {
		return step.Image
	}
	image := testjob.Spec.Prepare.Image
	if image != "" {
		return image
	}
	return testjob.Spec.Image
}

func (r *TestJobRunner) prepareWorkingDir(stepIdx int, testjob TestJob) string {
	step := testjob.Spec.Prepare.Steps[stepIdx]
	if step.Workdir != "" {
		return step.Workdir
	}
	return r.sharedVolumeMount().MountPath
}

func (r *TestJobRunner) prepareEnv(stepIdx int, testjob TestJob) []apiv1.EnvVar {
	step := testjob.Spec.Prepare.Steps[stepIdx]
	env := step.Env
	env = append(env, testjob.Spec.Env...)
	return env
}

func (r *TestJobRunner) enabledPrepareCheckout(testjob TestJob) bool {
	checkout := testjob.Spec.Prepare.Checkout
	if checkout != nil && !(*checkout) {
		return false
	}
	return true
}

func (r *TestJobRunner) enabledCheckout(testjob TestJob) bool {
	checkout := testjob.Spec.Checkout
	if checkout != nil && !(*checkout) {
		return false
	}
	return true
}

func (r *TestJobRunner) prepare(ctx context.Context, testjob TestJob) error {
	if len(testjob.Spec.Prepare.Steps) == 0 {
		return nil
	}
	var containers []apiv1.Container
	if r.enabledPrepareCheckout(testjob) {
		containers = r.initContainers(testjob)
	}
	fmt.Println("run prepare")
	for stepIdx, step := range testjob.Spec.Prepare.Steps {
		image := r.prepareImage(stepIdx, testjob)
		cmd, args := r.command(step.Command)
		volumeMount := r.sharedVolumeMount()
		containers = append(containers, apiv1.Container{
			Name:       step.Name,
			Image:      image,
			Command:    cmd,
			Args:       args,
			WorkingDir: r.prepareWorkingDir(stepIdx, testjob),
			VolumeMounts: []apiv1.VolumeMount{
				volumeMount,
			},
			Env: r.prepareEnv(stepIdx, testjob),
		})
	}
	lastContainer := containers[len(containers)-1]
	job, err := kubejob.NewJobBuilder(r.Clientset, testjob.Namespace).
		BuildWithJob(&batchv1.Job{
			Spec: batchv1.JobSpec{
				Template: apiv1.PodTemplateSpec{
					Spec: apiv1.PodSpec{
						Volumes: []apiv1.Volume{
							r.sharedVolume(),
						},
						InitContainers:   containers[:len(containers)-1],
						Containers:       []apiv1.Container{lastContainer},
						ImagePullSecrets: testjob.Spec.ImagePullSecrets,
					},
				},
			},
		})
	if err != nil {
		return err
	}
	if r.logger != nil {
		job.SetLogger(r.logger)
	}
	return job.Run(ctx)
}

func (r *TestJobRunner) newJobForTesting(testjob TestJob, containers []apiv1.Container) (*kubejob.Job, error) {
	var initContainers []apiv1.Container
	if r.enabledCheckout(testjob) {
		initContainers = r.initContainers(testjob)
	}
	return kubejob.NewJobBuilder(r.Clientset, testjob.Namespace).
		BuildWithJob(&batchv1.Job{
			Spec: batchv1.JobSpec{
				Template: apiv1.PodTemplateSpec{
					Spec: apiv1.PodSpec{
						Volumes: []apiv1.Volume{
							r.sharedVolume(),
						},
						InitContainers:   initContainers,
						Containers:       containers,
						ImagePullSecrets: testjob.Spec.ImagePullSecrets,
					},
				},
			},
		})
}

func (r *TestJobRunner) run(ctx context.Context, testjob TestJob) error {
	job, err := r.newJobForTesting(testjob, []apiv1.Container{r.testjobToContainer(testjob)})
	if err != nil {
		return err
	}
	if r.logger != nil {
		job.SetLogger(r.logger)
	}
	if r.disabledPrepareLog {
		job.DisableInitContainerLog()
	}
	if r.disabledCommandLog {
		job.DisableCommandLog()
	}
	if err := job.Run(ctx); err != nil {
		var failedJob *kubejob.FailedJob
		if xerrors.As(err, &failedJob) {
			return ErrFailedTestJob
		}
		log.Printf(err.Error())
		return ErrFailedTestJob
	}
	return nil
}

func (r *TestJobRunner) runDistributedTest(ctx context.Context, testjob TestJob) error {
	list, err := r.testList(ctx, testjob)
	if err != nil {
		return err
	}
	plan := r.plan(testjob, list)

	failedTestCommands := []*command{}

	var (
		mu         sync.Mutex
		lastPodIdx int
	)
	containerNameToLogMap := map[string][]string{}
	podNameToIndexMap := map[string]int{}
	logger := func(log *kubejob.ContainerLog) {
		mu.Lock()
		defer mu.Unlock()

		name := log.Container.Name
		if log.IsFinished {
			cmd, _ := r.containerNameToCommandMap.Load(name)
			logs, exists := containerNameToLogMap[name]
			if exists {
				podName := log.Pod.Name
				idx, exists := podNameToIndexMap[podName]
				if !exists {
					idx = lastPodIdx
					podNameToIndexMap[log.Pod.Name] = lastPodIdx
					lastPodIdx++
				}
				fmt.Fprintf(os.Stderr, "[POD %d] TEST=%s %s", idx, cmd.(*command).test, testjob.Spec.Command)
				for _, log := range logs {
					fmt.Fprintf(os.Stderr, "[POD %d] %s", idx, log)
				}
				fmt.Fprintf(os.Stderr, "\n")
			}
			delete(containerNameToLogMap, name)
		} else {
			value, exists := containerNameToLogMap[name]
			logs := []string{}
			if exists {
				logs = value
			}
			logs = append(logs, log.Log)
			containerNameToLogMap[name] = logs
		}
	}

	var eg errgroup.Group
	for _, tests := range plan {
		tests := tests
		commands := r.testsToCommands(testjob, tests)
		eg.Go(func() error {
			commands, err := r.runTests(ctx, testjob, logger, commands)
			if err != nil {
				return xerrors.Errorf("failed to runTests: %w", err)
			}
			if len(commands) > 0 {
				failedTestCommands = append(failedTestCommands, commands...)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return xerrors.Errorf("failed to distributed test job: %w", err)
	}

	if len(failedTestCommands) > 0 {
		if !testjob.Spec.DistributedTest.Retest {
			return ErrFailedTestJob
		}
		fmt.Println("start retest....")
		tests := []string{}
		for _, command := range failedTestCommands {
			tests = append(tests, command.test)
		}
		concatedTests := strings.Join(tests, testjob.Spec.DistributedTest.RetestDelimiter)
		command, args := r.command(testjob.Spec.Command)
		cmd := r.testCommand(command, args, concatedTests)
		failedTests, err := r.runTests(ctx, testjob, logger, commands{cmd})
		if err != nil {
			return xerrors.Errorf("failed test: %w", err)
		}
		if len(failedTests) > 0 {
			return ErrFailedTestJob
		}
	}
	return nil
}

type command struct {
	cmd       []string
	args      []string
	test      string
	container string
}

func (c *command) value() string {
	cmd := []string{}
	cmd = append(cmd, c.cmd...)
	cmd = append(cmd, c.args...)
	return strings.Join(cmd, " ")
}

type commands []*command

func (c commands) commandValueMap() map[string]*command {
	m := map[string]*command{}
	for _, cc := range c {
		m[cc.value()] = cc
	}
	return m
}

func (r *TestJobRunner) testCommand(cmd []string, args []string, test string) *command {
	return &command{
		cmd:  cmd,
		args: args,
		test: test,
	}
}

func (r *TestJobRunner) testsToCommands(job TestJob, tests []string) []*command {
	c, args := r.command(job.Spec.Command)
	commands := []*command{}
	for _, test := range tests {
		cmd := r.testCommand(c, args, test)
		commands = append(commands, cmd)
	}
	return commands
}

func (r *TestJobRunner) workingDir(testjob TestJob) string {
	if testjob.Spec.Workdir != "" {
		return testjob.Spec.Workdir
	}
	return r.sharedVolumeMount().MountPath
}

func (r *TestJobRunner) testjobToContainer(testjob TestJob) apiv1.Container {
	cmd, args := r.command(testjob.Spec.Command)
	volumeMount := r.sharedVolumeMount()
	return apiv1.Container{
		Image:      testjob.Spec.Image,
		Command:    cmd,
		Args:       args,
		WorkingDir: r.workingDir(testjob),
		VolumeMounts: []apiv1.VolumeMount{
			volumeMount,
		},
		Env: testjob.Spec.Env,
	}
}

func (r *TestJobRunner) testCommandToContainer(job TestJob, test *command) apiv1.Container {
	volumeMount := r.sharedVolumeMount()
	env := []apiv1.EnvVar{}
	env = append(env, job.Spec.Env...)
	env = append(env, apiv1.EnvVar{
		Name:  "TEST",
		Value: test.test,
	})
	return apiv1.Container{
		Image:      job.Spec.Image,
		Command:    test.cmd,
		Args:       test.args,
		WorkingDir: r.workingDir(job),
		VolumeMounts: []apiv1.VolumeMount{
			volumeMount,
		},
		Env: env,
	}
}

func (r *TestJobRunner) runTests(ctx context.Context, testjob TestJob, logger kubejob.Logger, testCommands commands) ([]*command, error) {
	concurrentNum := defaultConcurrentNum
	failedTestCommands := []*command{}
	commandValueMap := testCommands.commandValueMap()
	for i := 0; i < len(testCommands); i += concurrentNum {
		containers := []apiv1.Container{}
		for j := 0; j < concurrentNum; j++ {
			if i+j < len(testCommands) {
				containers = append(containers, r.testCommandToContainer(testjob, testCommands[i+j]))
			}
		}
		job, err := r.newJobForTesting(testjob, containers)
		if err != nil {
			return nil, err
		}
		for j := 0; j < concurrentNum; j++ {
			if i+j < len(testCommands) {
				containerName := job.Spec.Template.Spec.Containers[j].Name
				testCommands[i+j].container = containerName
				r.containerNameToCommandMap.Store(containerName, testCommands[i+j])
			}
		}
		job.DisableCommandLog()
		job.SetLogger(logger)
		if err := job.Run(ctx); err != nil {
			var failedJob *kubejob.FailedJob
			if xerrors.As(err, &failedJob) {
				for _, container := range failedJob.FailedContainers() {
					cmd := []string{}
					cmd = append(cmd, container.Command...)
					cmd = append(cmd, container.Args...)
					value := strings.Join(cmd, " ")
					command := commandValueMap[value]
					failedTestCommands = append(failedTestCommands, command)
				}
			} else {
				return nil, err
			}
		}
	}
	return failedTestCommands, nil
}

func (r *TestJobRunner) testList(ctx context.Context, testjob TestJob) ([]string, error) {
	distributedTest := testjob.Spec.DistributedTest

	listjob := testjob
	listjob.Spec.Command = distributedTest.ListCommand
	listjob.Spec.Prepare.Steps = []PrepareStepSpec{}
	listjob.Spec.DistributedTest = nil

	listJobRunner := NewTestJobRunner(r.Clientset)
	listJobRunner.DisablePrepareLog()
	listJobRunner.DisableCommandLog()

	var pattern *regexp.Regexp
	if distributedTest.Pattern != "" {
		reg, err := regexp.Compile(distributedTest.Pattern)
		if err != nil {
			return nil, err
		}
		pattern = reg
	}

	var b bytes.Buffer
	listJobRunner.SetLogger(func(log *kubejob.ContainerLog) {
		b.WriteString(log.Log)
	})
	if err := listJobRunner.Run(ctx, listjob); err != nil {
		return nil, err
	}
	result := b.String()
	if len(result) == 0 {
		return nil, xerrors.Errorf("could not find test list. list is empty")
	}
	delim := distributedTest.ListDelimiter
	if delim == "" {
		delim = "\n"
	}
	tests := []string{}
	list := strings.Split(result, delim)
	if pattern != nil {
		for _, name := range list {
			if pattern.MatchString(name) {
				tests = append(tests, name)
			}
		}
	} else {
		tests = list
	}
	if len(tests) == 0 {
		return nil, xerrors.Errorf("could not find test list. list is invalid %s", result)
	}
	return tests, nil
}

func (r *TestJobRunner) plan(job TestJob, list []string) [][]string {
	concurrent := job.Spec.DistributedTest.Concurrent

	if len(list) < concurrent {
		plan := make([][]string, len(list))
		for i := 0; i < len(list); i++ {
			plan[i] = []string{list[i]}
		}
		return plan
	}
	testNum := len(list) / concurrent
	lastIdx := concurrent - 1
	plan := [][]string{}
	sum := 0
	for i := 0; i < concurrent; i++ {
		if i == lastIdx {
			plan = append(plan, list[sum:])
		} else {
			plan = append(plan, list[sum:sum+testNum])
		}
		sum += testNum
	}
	return plan
}
