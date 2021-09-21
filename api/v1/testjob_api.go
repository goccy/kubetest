package v1

/*
import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/kubejob"
	"golang.org/x/xerrors"
	apiv1 "k8s.io/api/core/v1"
)

const (
	repoVolumeName    = "repo"
	archiveVolumeName = "archive"
	accessTokenEnv    = "ACCESS_TOKEN"
	defaultGitImage   = "alpine/git"
	defaultBranch     = "master"
	defaultListDelim  = "\n"
	listContainerName = "list"
	listJobLabel      = "kubetest.io/listjob"
	testJobLabel      = "kubetest.io/testjob"
	testsAnnotation   = "kubetest.io/tests"
)

var (
	defaultWorkspacePath = filepath.Join("/", "git", "workspace")
	repoMountPath        = filepath.Join("/", "git", "repo")
	archiveMountPath     = filepath.Join("/", "git", "archive")
)

func (j TestJob) existsPrepareSteps() bool {
	return len(j.Spec.PreSteps) > 0
}

func (j TestJob) enabledDistributedTest() bool {
	return j.Spec.DistributedTest != nil
}

func (j TestJob) enabledRetest() bool {
	return j.Spec.DistributedTest.Retest
}

func (j TestJob) listDelim() string {
	if j.Spec.DistributedTest == nil {
		return ""
	}
	delim := j.Spec.DistributedTest.List.Delimiter
	if delim != "" {
		return delim
	}
	return defaultListDelim
}

func (j TestJob) listNames() []string {
	if j.Spec.DistributedTest == nil {
		return nil
	}
	return j.Spec.DistributedTest.List.Names
}

func (j TestJob) workingDir(c apiv1.Container) string {
	if c.WorkingDir != "" {
		return c.WorkingDir
	}
	return j.workspacePath()
}

func (j TestJob) enabledCheckout() bool {
	checkout := j.Spec.Git.Checkout
	if checkout != nil && !(*checkout) {
		return false
	}
	return true
}

func (j TestJob) workspacePath() string {
	checkoutDir := j.checkoutDir()
	if checkoutDir != "" {
		return checkoutDir
	}
	return defaultWorkspacePath
}

func (j TestJob) repoVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: repoVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
}

func (j TestJob) workspaceVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      repoVolumeName,
		MountPath: j.workspacePath(),
	}
}

func (j TestJob) repoVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      repoVolumeName,
		MountPath: repoMountPath,
	}
}

func (j TestJob) archiveVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: archiveVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
}

func (j TestJob) archiveVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      archiveVolumeName,
		MountPath: archiveMountPath,
	}
}

func (j TestJob) gitCloneURL(requiredToken bool) string {
	if requiredToken {
		return fmt.Sprintf("https://x-access-token:$(%s)@%s.git", accessTokenEnv, j.repo())
	}
	return fmt.Sprintf("https://%s.git", j.repo())
}

func (j TestJob) gitCloneCommand(cloneURL string) ([]string, []string) {
	branch := j.branch()
	if branch != "" {
		return []string{"git"}, []string{"clone", "-b", branch, cloneURL, j.workspacePath()}
	}
	return []string{"git"}, []string{"clone", cloneURL, j.workspacePath()}
}

func (j TestJob) gitCloneContainer(tokenSecret *apiv1.SecretKeySelector) apiv1.Container {
	cmd, args := j.gitCloneCommand(j.gitCloneURL(tokenSecret != nil))
	env := []apiv1.EnvVar{}
	if tokenSecret != nil {
		env = append(env, apiv1.EnvVar{
			Name: accessTokenEnv, ValueFrom: &apiv1.EnvVarSource{SecretKeyRef: tokenSecret},
		})
	}
	return apiv1.Container{
		Name:         "kubetest-init-clone",
		Image:        j.gitImage(),
		Command:      cmd,
		Args:         args,
		Env:          env,
		VolumeMounts: []apiv1.VolumeMount{j.workspaceVolumeMount()},
	}
}

func (j TestJob) gitSwitchContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-switch",
		Image:        j.gitImage(),
		WorkingDir:   j.workspacePath(),
		Command:      []string{"git"},
		Args:         []string{"checkout", "--detach", j.rev()},
		VolumeMounts: []apiv1.VolumeMount{j.workspaceVolumeMount()},
	}
}

func (j TestJob) gitConfigUserEmailContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-git-config-user-email",
		Image:        j.gitImage(),
		WorkingDir:   j.workspacePath(),
		Command:      []string{"git"},
		Args:         []string{"config", "user.email", "anonymous@kubetest.com"},
		VolumeMounts: []apiv1.VolumeMount{j.workspaceVolumeMount()},
	}
}

func (j TestJob) gitConfigUserNameContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-git-config-user-name",
		Image:        j.gitImage(),
		WorkingDir:   j.workspacePath(),
		Command:      []string{"git"},
		Args:         []string{"config", "user.name", "anonymous"},
		VolumeMounts: []apiv1.VolumeMount{j.workspaceVolumeMount()},
	}
}

func (j TestJob) gitMergeContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-merge",
		Image:        j.gitImage(),
		WorkingDir:   j.workspacePath(),
		Command:      []string{"git"},
		Args:         []string{"pull", "origin", j.baseBranch()},
		VolumeMounts: []apiv1.VolumeMount{j.workspaceVolumeMount()},
	}
}

func (j TestJob) packRepoContainer() apiv1.Container {
	base, target := filepath.Split(j.workspacePath())
	return apiv1.Container{
		Name:       "kubetest-pack-repo",
		Image:      j.gitImage(),
		WorkingDir: base,
		Command:    []string{"tar"},
		Args: []string{
			"-czf",
			filepath.Join(archiveMountPath, "repo.tar.gz"),
			target,
		},
		VolumeMounts: []apiv1.VolumeMount{
			j.workspaceVolumeMount(),
			j.archiveVolumeMount(),
		},
	}
}

func (j TestJob) unpackRepoCommand() []string {
	base, _ := filepath.Split(j.workspacePath())
	return []string{
		"tar",
		"-zxvf",
		filepath.Join(archiveMountPath, "repo.tar.gz"),
		"-C",
		base,
	}
}

func (j TestJob) initContainers(tokenSecret *apiv1.SecretKeySelector) []apiv1.Container {
	containers := []apiv1.Container{}
	branch := j.branch()
	if branch == "" && j.rev() == "" {
		branch = defaultBranch
	}
	if branch != "" {
		containers = append(containers, j.gitCloneContainer(tokenSecret))
	} else {
		containers = append(containers, j.gitCloneContainer(tokenSecret), j.gitSwitchContainer())
	}
	if j.baseBranch() != "" {
		containers = append(containers,
			j.gitConfigUserEmailContainer(),
			j.gitConfigUserNameContainer(),
			j.gitMergeContainer(),
		)
	}
	return containers
}

func (j TestJob) testInitContainers(tokenSecret *apiv1.SecretKeySelector) []apiv1.Container {
	if j.enabledCheckout() {
		return append(j.initContainers(tokenSecret), j.Spec.Template.Spec.InitContainers...)
	}
	return j.Spec.Template.Spec.InitContainers
}

func (j TestJob) testContainerName() string {
	if j.Spec.DistributedTest == nil {
		return ""
	}
	return j.Spec.DistributedTest.ContainerName
}

func (j TestJob) testContainers(extraContainers ...apiv1.Container) []apiv1.Container {
	testContainers := []apiv1.Container{}
	isDistributedTest := len(extraContainers) > 0
	for _, container := range j.Spec.Template.Spec.Containers {
		container := container
		if isDistributedTest && container.Name == j.testContainerName() {
			// skip default test container
			continue
		}
		container.VolumeMounts = append(container.VolumeMounts, j.archiveVolumeMount())
		if j.Spec.DistributedTest == nil {
			container.VolumeMounts = append(container.VolumeMounts, j.workspaceVolumeMount())
		}
		testContainers = append(testContainers, container)
	}
	for _, container := range extraContainers {
		container := container
		container.VolumeMounts = append(container.VolumeMounts, j.archiveVolumeMount())
		if j.Spec.DistributedTest == nil {
			container.VolumeMounts = append(container.VolumeMounts, j.workspaceVolumeMount())
		}
		testContainers = append(testContainers, container)
	}
	return testContainers
}

func (j TestJob) defaultTestContainer() (apiv1.Container, error) {
	name := j.testContainerName()
	for _, container := range j.Spec.Template.Spec.Containers {
		container := container
		if container.Name == name {
			c := container.DeepCopy()
			return *c, nil
		}
	}
	return apiv1.Container{}, xerrors.Errorf("cannot find container for running test by name: %s", name)
}

func (j TestJob) testContainerByName(name string) (apiv1.Container, error) {
	c, err := j.defaultTestContainer()
	if err != nil {
		return apiv1.Container{}, xerrors.Errorf("failed to create default test container: %w", err)
	}
	c.WorkingDir = j.workingDir(c)
	c.Name = "" // remove default test container name
	c.Env = append(c.Env, apiv1.EnvVar{
		Name:  "TEST",
		Value: name,
	})
	return c, nil
}

func (j TestJob) createJobTemplate(tokenSecret *apiv1.SecretKeySelector, extraContainers ...apiv1.Container) apiv1.PodTemplateSpec {
	template := j.Spec.Template // copy template
	template.Spec.InitContainers = j.testInitContainers(tokenSecret)
	template.Spec.Containers = j.testContainers(extraContainers...)
	template.Spec.Volumes = append(
		template.Spec.Volumes,
		j.repoVolume(),
		j.archiveVolume(),
	)

	newLabels := map[string]string{}
	for k, v := range template.ObjectMeta.Labels {
		newLabels[k] = v
	}
	template.ObjectMeta.Labels = newLabels

	newAnnotations := map[string]string{}
	for k, v := range template.ObjectMeta.Annotations {
		newAnnotations[k] = v
	}
	template.ObjectMeta.Annotations = newAnnotations

	return template
}

func (j TestJob) createListJobTemplate(tokenSecret *apiv1.SecretKeySelector) (apiv1.PodTemplateSpec, error) {
	c, err := j.defaultTestContainer()
	if err != nil {
		return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to create default test container: %w", err)
	}
	listSpec := j.Spec.DistributedTest.List
	c.Name = listContainerName
	c.Command = listSpec.Command
	c.Args = listSpec.Args
	c.WorkingDir = j.workingDir(c)
	template := j.createJobTemplate(tokenSecret, c)
	if j.enabledCheckout() {
		template.Spec.InitContainers = append(template.Spec.InitContainers, j.packRepoContainer())
	}
	template.ObjectMeta.Labels[listJobLabel] = fmt.Sprint(true)
	return template, nil
}

func (j TestJob) createTestJobTemplate(tokenSecret *apiv1.SecretKeySelector, tests []string) (apiv1.PodTemplateSpec, error) {
	containers := []apiv1.Container{}
	for _, test := range tests {
		test := test
		container, err := j.testContainerByName(test)
		if err != nil {
			return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to create test container by name: %w", err)
		}
		containers = append(containers, container)
	}
	template := j.createJobTemplate(tokenSecret, containers...)
	template.ObjectMeta.Labels[testJobLabel] = fmt.Sprint(true)

	encodedTests, err := json.Marshal(tests)
	if err != nil {
		return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to encode tests: %w", err)
	}
	template.ObjectMeta.Annotations[testsAnnotation] = string(encodedTests)

	c, err := j.defaultTestContainer()
	if err != nil {
		return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to create default test container: %w", err)
	}
	if j.Spec.DistributedTest != nil && j.enabledCheckout() {
		template.Spec.InitContainers = append(template.Spec.InitContainers, j.packRepoContainer())
	}
	return template, nil
}

func (j TestJob) filterTestExecutors(executors []*kubejob.JobExecutor) []*kubejob.JobExecutor {
	testExecutors := []*kubejob.JobExecutor{}
	for _, executor := range executors {
		executor := executor
		if j.testNameByExecutor(executor) != "" {
			testExecutors = append(testExecutors, executor)
		}
	}
	return testExecutors
}

func (j TestJob) filterSidecarExecutors(executors []*kubejob.JobExecutor) []*kubejob.JobExecutor {
	sidecarExecutors := []*kubejob.JobExecutor{}
	for _, executor := range executors {
		executor := executor
		if j.testNameByExecutor(executor) == "" {
			sidecarExecutors = append(sidecarExecutors, executor)
		}
	}
	return sidecarExecutors
}

func (j TestJob) concurrentNum(executorNum int) int {
	concurrent := j.Spec.DistributedTest.MaxConcurrentNumPerPod
	if concurrent <= 0 {
		return executorNum
	}
	if concurrent > executorNum {
		return executorNum
	}
	return concurrent
}

func (j TestJob) testNameByExecutor(executor *kubejob.JobExecutor) string {
	for _, env := range executor.Container.Env {
		if env.Name == "TEST" {
			return env.Value
		}
	}
	return ""
}

func (j TestJob) testCommand(testName string) (string, error) {
	c, err := j.defaultTestContainer()
	if err != nil {
		return "", xerrors.Errorf("failed to create default test container: %w", err)
	}
	cmd := []string{}
	cmd = append(cmd, c.Command...)
	cmd = append(cmd, c.Args...)
	return fmt.Sprintf("TEST=%s; %s", testName, strings.Join(cmd, " ")), nil
}

func (j TestJob) schedule(executors []*kubejob.JobExecutor) [][]*kubejob.JobExecutor {
	executorNum := len(executors)
	concurrent := j.concurrentNum(executorNum)

	scheduledExecutors := [][]*kubejob.JobExecutor{}
	for i := 0; i < executorNum; i += concurrent {
		start := i
		end := i + concurrent
		if end > executorNum {
			end = executorNum
		}
		scheduledExecutors = append(scheduledExecutors, executors[start:end])
	}
	return scheduledExecutors
}

*/
