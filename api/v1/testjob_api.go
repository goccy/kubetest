package v1

import (
	"encoding/base64"
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
	sharedVolumeName  = "repo"
	oauthTokenEnv     = "OAUTH_TOKEN"
	defaultGitImage   = "alpine/git"
	defaultBranch     = "master"
	defaultListDelim  = "\n"
	listContainerName = "list"
	listJobLabel      = "kubetest.io/listjob"
	testJobLabel      = "kubetest.io/testjob"
	testsAnnotation   = "kubetest.io/tests"
)

var (
	defaultVolumeMountPath = filepath.Join("/", "git", "workspace")
)

func (j TestJob) existsPrepareSteps() bool {
	return len(j.Spec.Prepare.Steps) > 0
}

func (j TestJob) prepareImage(step int) string {
	if len(j.Spec.Prepare.Steps) <= step {
		return ""
	}
	image := j.Spec.Prepare.Steps[step].Image
	if image != "" {
		return image
	}
	return j.Spec.Prepare.Image
}

func (j TestJob) prepareWorkingDir(step int) string {
	if len(j.Spec.Prepare.Steps) <= step {
		return ""
	}
	dir := j.Spec.Prepare.Steps[step].Workdir
	if dir != "" {
		return dir
	}
	return j.volumeMountPath()
}

func (j TestJob) prepareEnv(step int) []apiv1.EnvVar {
	if len(j.Spec.Prepare.Steps) <= step {
		return nil
	}
	return j.Spec.Prepare.Steps[step].Env
}

func (j TestJob) enabledPrepareCheckout() bool {
	checkout := j.Spec.Prepare.Checkout
	if checkout != nil && !(*checkout) {
		return false
	}
	return true
}

func (j TestJob) enabledDistributedTest() bool {
	return j.Spec.DistributedTest != nil
}

func (j TestJob) enabledRetest() bool {
	return j.Spec.DistributedTest.Retest
}

func (j TestJob) gitToken() *TestJobToken {
	return j.Spec.Git.Token
}

func (j TestJob) checkoutDir() string {
	return j.Spec.Git.CheckoutDir
}

func (j TestJob) repo() string {
	return j.Spec.Git.Repo
}

func (j TestJob) branch() string {
	return j.Spec.Git.Branch
}

func (j TestJob) baseBranch() string {
	return j.Spec.Git.Merge.Base
}

func (j TestJob) rev() string {
	return j.Spec.Git.Rev
}

func (j TestJob) gitImage() string {
	image := j.Spec.Git.Image
	if image != "" {
		return image
	}
	return defaultGitImage
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

func (j TestJob) workingDir(c apiv1.Container) string {
	if c.WorkingDir != "" {
		return c.WorkingDir
	}
	return j.volumeMountPath()
}

func (j TestJob) enabledCheckout() bool {
	checkout := j.Spec.Git.Checkout
	if checkout != nil && !(*checkout) {
		return false
	}
	return true
}

func (j TestJob) volumeMountPath() string {
	checkoutDir := j.checkoutDir()
	if checkoutDir != "" {
		return checkoutDir
	}
	return defaultVolumeMountPath
}

func (j TestJob) sharedVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: sharedVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
}

func (j TestJob) sharedVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      sharedVolumeName,
		MountPath: j.volumeMountPath(),
	}
}

func (j TestJob) gitCloneURL(token string) string {
	if token != "" {
		return fmt.Sprintf("https://$(%s)@%s.git", oauthTokenEnv, j.repo())
	}
	return fmt.Sprintf("https://%s.git", j.repo())
}

func (j TestJob) gitCloneCommand(cloneURL string) ([]string, []string) {
	mountPath := j.volumeMountPath()
	branch := j.branch()
	if branch != "" {
		return []string{"git"}, []string{"clone", "-b", branch, cloneURL, mountPath}
	}
	return []string{"git"}, []string{"clone", cloneURL, mountPath}
}

func (j TestJob) gitCloneContainer(token string) apiv1.Container {
	cmd, args := j.gitCloneCommand(j.gitCloneURL(token))
	return apiv1.Container{
		Name:         "kubetest-init-clone",
		Image:        j.gitImage(),
		Command:      cmd,
		Args:         args,
		Env:          []apiv1.EnvVar{{Name: oauthTokenEnv, Value: token}},
		VolumeMounts: []apiv1.VolumeMount{j.sharedVolumeMount()},
	}
}

func (j TestJob) gitSwitchContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-switch",
		Image:        j.gitImage(),
		WorkingDir:   j.volumeMountPath(),
		Command:      []string{"git"},
		Args:         []string{"checkout", "--detach", j.rev()},
		VolumeMounts: []apiv1.VolumeMount{j.sharedVolumeMount()},
	}
}

func (j TestJob) gitConfigUserEmailContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-git-config-user-email",
		Image:        j.gitImage(),
		WorkingDir:   j.volumeMountPath(),
		Command:      []string{"git"},
		Args:         []string{"config", "user.email", "anonymous@kubetest.com"},
		VolumeMounts: []apiv1.VolumeMount{j.sharedVolumeMount()},
	}
}

func (j TestJob) gitConfigUserNameContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-git-config-user-name",
		Image:        j.gitImage(),
		WorkingDir:   j.volumeMountPath(),
		Command:      []string{"git"},
		Args:         []string{"config", "user.name", "anonymous"},
		VolumeMounts: []apiv1.VolumeMount{j.sharedVolumeMount()},
	}
}

func (j TestJob) gitMergeContainer() apiv1.Container {
	return apiv1.Container{
		Name:         "kubetest-init-merge",
		Image:        j.gitImage(),
		WorkingDir:   j.volumeMountPath(),
		Command:      []string{"git"},
		Args:         []string{"pull", "origin", j.baseBranch()},
		VolumeMounts: []apiv1.VolumeMount{j.sharedVolumeMount()},
	}
}

func (j TestJob) initContainers(token string) []apiv1.Container {
	containers := []apiv1.Container{}
	branch := j.branch()
	if branch == "" && j.rev() == "" {
		branch = defaultBranch
	}
	if branch != "" {
		containers = append(containers, j.gitCloneContainer(token))
	} else {
		containers = append(containers, j.gitCloneContainer(token), j.gitSwitchContainer())
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

func (j TestJob) testInitContainers(token string) []apiv1.Container {
	if j.enabledCheckout() {
		return append(j.initContainers(token), j.Spec.Template.Spec.InitContainers...)
	}
	return j.Spec.Template.Spec.InitContainers
}

func (j TestJob) prepareInitContainers(token string) []apiv1.Container {
	if j.enabledPrepareCheckout() {
		return j.initContainers(token)
	}
	return nil
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
		if isDistributedTest && container.Name == j.testContainerName() {
			// skip default test container
			continue
		}
		container.VolumeMounts = append(container.VolumeMounts, j.sharedVolumeMount())
		testContainers = append(testContainers, container)
	}
	for _, container := range extraContainers {
		container.VolumeMounts = append(container.VolumeMounts, j.sharedVolumeMount())
		testContainers = append(testContainers, container)
	}
	return testContainers
}

func (j TestJob) defaultTestContainer() (apiv1.Container, error) {
	name := j.testContainerName()
	for _, container := range j.Spec.Template.Spec.Containers {
		if container.Name == name {
			return container, nil
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

func (j TestJob) createJobTemplate(token string, extraContainers ...apiv1.Container) apiv1.PodTemplateSpec {
	template := j.Spec.Template // copy template
	template.Spec.InitContainers = j.testInitContainers(token)
	template.Spec.Containers = j.testContainers(extraContainers...)
	template.Spec.Volumes = append(template.Spec.Volumes, j.sharedVolume())

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

func (j TestJob) createPrepareJobTemplate(token string) (apiv1.PodTemplateSpec, error) {
	if len(j.Spec.Prepare.Steps) == 0 {
		return apiv1.PodTemplateSpec{}, nil
	}
	containers := j.prepareInitContainers(token)
	for stepIdx, step := range j.Spec.Prepare.Steps {
		cmd, args := j.escapedCommand(step.Command)
		containers = append(containers, apiv1.Container{
			Name:       step.Name,
			Image:      j.prepareImage(stepIdx),
			Command:    cmd,
			Args:       args,
			WorkingDir: j.prepareWorkingDir(stepIdx),
			VolumeMounts: []apiv1.VolumeMount{
				j.sharedVolumeMount(),
			},
			Env: j.prepareEnv(stepIdx),
		})
	}
	lastContainer := containers[len(containers)-1]
	initContainers := containers[:len(containers)-1]
	return apiv1.PodTemplateSpec{
		Spec: apiv1.PodSpec{
			Volumes: []apiv1.Volume{
				j.sharedVolume(),
			},
			InitContainers:   initContainers,
			Containers:       []apiv1.Container{lastContainer},
			ImagePullSecrets: j.Spec.Template.Spec.ImagePullSecrets,
		},
	}, nil
}

func (j TestJob) createListJobTemplate(token string) (apiv1.PodTemplateSpec, error) {
	c, err := j.defaultTestContainer()
	if err != nil {
		return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to create default test container: %w", err)
	}
	listSpec := j.Spec.DistributedTest.List
	c.Name = listContainerName
	c.Command = listSpec.Command
	c.Args = listSpec.Args
	c.WorkingDir = j.workingDir(c)
	template := j.createJobTemplate(token, c)
	template.ObjectMeta.Labels[listJobLabel] = fmt.Sprint(true)
	return template, nil
}

func (j TestJob) escapedCommand(cmd Command) ([]string, []string) {
	e := base64.StdEncoding.EncodeToString([]byte(string(cmd)))
	return []string{"sh"}, []string{"-c", fmt.Sprintf("echo %s | base64 -d | sh", e)}
}

func (j TestJob) createTestJobTemplate(token string, tests []string) (apiv1.PodTemplateSpec, error) {
	containers := []apiv1.Container{}
	for _, test := range tests {
		container, err := j.testContainerByName(test)
		if err != nil {
			return apiv1.PodTemplateSpec{}, xerrors.Errorf("failed to create test container by name: %w", err)
		}
		containers = append(containers, container)
	}
	template := j.createJobTemplate(token, containers...)
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
	for _, cache := range j.Spec.DistributedTest.Cache {
		cmd, args := j.escapedCommand(cache.Command)
		volumeMounts := append(c.VolumeMounts, j.sharedVolumeMount(), apiv1.VolumeMount{
			Name:      cache.Name,
			MountPath: cache.Path,
		})
		cacheContainer := apiv1.Container{
			Name:         cache.Name,
			Image:        c.Image,
			Command:      cmd,
			Args:         args,
			WorkingDir:   j.workingDir(c),
			VolumeMounts: volumeMounts,
			Env:          c.Env,
		}
		template.Spec.Volumes = append(template.Spec.Volumes, apiv1.Volume{
			Name: cache.Name,
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		})
		template.Spec.InitContainers = append(template.Spec.InitContainers, cacheContainer)
	}
	return template, nil
}

func (j TestJob) filterTestExecutors(executors []*kubejob.JobExecutor) []*kubejob.JobExecutor {
	name := j.Spec.DistributedTest.ContainerName
	testExecutors := []*kubejob.JobExecutor{}
	for _, executor := range executors {
		if executor.Container.Name == name {
			testExecutors = append(testExecutors, executor)
		}
	}
	return testExecutors
}

func (j TestJob) filterSidecarExecutors(executors []*kubejob.JobExecutor) []*kubejob.JobExecutor {
	name := j.Spec.DistributedTest.ContainerName
	sidecarExecutors := []*kubejob.JobExecutor{}
	for _, executor := range executors {
		if executor.Container.Name != name {
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
	return fmt.Sprintf("TEST=%s: %s", testName, strings.Join(cmd, " ")), nil
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

func (j TestJob) listPattern() (*regexp.Regexp, error) {
	if j.Spec.DistributedTest == nil {
		return nil, xerrors.Errorf("failed to create list pattern. Spec.DistributedTest is nil")
	}
	listPattern := j.Spec.DistributedTest.List.Pattern
	if listPattern == "" {
		return nil, nil
	}
	reg, err := regexp.Compile(listPattern)
	if err != nil {
		return nil, xerrors.Errorf("failed to compile pattern for distributed testing: %w", err)
	}
	return reg, nil
}

func (j TestJob) splitTest(src string) ([]string, error) {
	pattern, err := j.listPattern()
	if err != nil {
		return nil, xerrors.Errorf("failed to get pattern for list: %w", err)
	}

	delim := j.listDelim()
	list := strings.Split(src, delim)
	if pattern == nil {
		return list, nil
	}

	tests := []string{}
	for _, name := range list {
		if pattern.MatchString(name) {
			tests = append(tests, name)
		}
	}
	return tests, nil
}

func (j TestJob) plan(tests []string) [][]string {
	if j.Spec.DistributedTest == nil {
		return [][]string{tests}
	}
	maxContainers := j.Spec.DistributedTest.MaxContainersPerPod

	if len(tests) <= maxContainers {
		return [][]string{tests}
	}
	concurrent := len(tests) / maxContainers
	plan := [][]string{}
	sum := 0
	for i := 0; i <= concurrent; i++ {
		if i == concurrent {
			plan = append(plan, tests[sum:])
		} else {
			plan = append(plan, tests[sum:sum+maxContainers])
		}
		sum += maxContainers
	}
	return plan
}

func (j TestJob) validate() error {
	if err := j.validateArtifacts(); err != nil {
		return xerrors.Errorf("invalid artifacts: %w", err)
	}
	return nil
}

func (j TestJob) validateArtifacts() error {
	if j.Spec.DistributedTest == nil {
		return nil
	}
	if j.Spec.DistributedTest.Artifacts == nil {
		return nil
	}
	artifacts := j.Spec.DistributedTest.Artifacts
	if len(artifacts.Paths) == 0 {
		return xerrors.Errorf("failed to find any paths in artifacts")
	}
	if artifacts.Output.Path == "" {
		return xerrors.Errorf("failed to find output path in artifacts")
	}
	return nil
}
