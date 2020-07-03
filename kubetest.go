package kubetest

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/goccy/kubejob"
	"golang.org/x/xerrors"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	gitImageName     = "alpine/git"
	githubTokenEnv   = "GITHUB_TOKEN"
	sharedVolumeName = "repo"
)

type tokenFromSecret struct {
	name string
	key  string
}

type TestJobBuilder struct {
	clientset       *kubernetes.Clientset
	namespace       string
	user            string
	repo            string
	branch          string
	rev             string
	image           string
	cmd             []string
	token           string
	tokenFromSecret *tokenFromSecret
}

func NewTestJobBuilder(clientset *kubernetes.Clientset, namespace string) *TestJobBuilder {
	return &TestJobBuilder{
		clientset: clientset,
		namespace: namespace,
	}
}

func (b *TestJobBuilder) SetUser(user string) *TestJobBuilder {
	b.user = user
	return b
}

func (b *TestJobBuilder) SetRepo(repo string) *TestJobBuilder {
	if strings.Contains(repo, "/") {
		splitted := strings.Split(repo, "/")
		if len(splitted) != 2 {
			return b
		}
		b.user = splitted[0]
		b.repo = splitted[1]
	} else {
		b.repo = repo
	}
	return b
}

func (b *TestJobBuilder) SetBranch(branch string) *TestJobBuilder {
	b.branch = branch
	return b
}

func (b *TestJobBuilder) SetRev(rev string) *TestJobBuilder {
	b.rev = rev
	return b
}

func (b *TestJobBuilder) SetImage(image string) *TestJobBuilder {
	b.image = image
	return b
}

func (b *TestJobBuilder) SetCommand(cmd []string) *TestJobBuilder {
	b.cmd = cmd
	return b
}

func (b *TestJobBuilder) SetToken(token string) *TestJobBuilder {
	b.token = token
	return b
}

func (b *TestJobBuilder) SetTokenFromSecret(name string, key string) *TestJobBuilder {
	b.tokenFromSecret = &tokenFromSecret{name: name, key: key}
	return b
}

func (t *TestJobBuilder) authToken() (string, error) {
	if t.token != "" {
		return t.token, nil
	}
	token := t.tokenFromSecret
	if token == nil {
		return "", nil
	}
	secret, err := t.clientset.CoreV1().Secrets(t.namespace).Get(token.name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	data := strings.TrimSpace(string(secret.Data[token.key]))
	return data, nil
}

func (b *TestJobBuilder) validate() error {
	if b.user == "" {
		return xerrors.Errorf("invalid testjob: user does not defined")
	}
	if b.repo == "" {
		return xerrors.Errorf("invalid testjob: repo does not defined")
	}
	if b.image == "" {
		return xerrors.Errorf("invalid testjob: image does not defined")
	}
	if len(b.cmd) == 0 {
		return xerrors.Errorf("invalid testjob: command does not defined")
	}
	if b.branch == "" && b.rev == "" {
		return xerrors.Errorf("invalid testjob: branch and revision does not defined")
	}
	return nil
}

func (b *TestJobBuilder) Build() (*TestJob, error) {
	if err := b.validate(); err != nil {
		return nil, xerrors.Errorf("failed to build testjob: %w", err)
	}
	token, err := b.authToken()
	if err != nil {
		return nil, xerrors.Errorf("failed to get auth token: %w", err)
	}
	return &TestJob{
		user:      b.user,
		repo:      b.repo,
		rev:       b.rev,
		branch:    b.branch,
		image:     b.image,
		cmd:       b.cmd,
		clientset: b.clientset,
		namespace: b.namespace,
		token:     token,
	}, nil
}

type TestJob struct {
	w                  io.Writer
	user               string
	repo               string
	rev                string
	branch             string
	image              string
	token              string
	cmd                []string
	clientset          *kubernetes.Clientset
	namespace          string
	disabledPrepareLog bool
	disabledCommandLog bool
}

func (t *TestJob) sharedVolume() apiv1.Volume {
	return apiv1.Volume{
		Name: sharedVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}
}

func (t *TestJob) sharedVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      sharedVolumeName,
		MountPath: filepath.Join("/", "git", "workspace"),
	}
}

func (t *TestJob) gitCloneContainer() apiv1.Container {
	var cloneURL string
	if t.token != "" {
		cloneURL = fmt.Sprintf("https://$(%s)@github.com/%s/%s.git", githubTokenEnv, t.user, t.repo)
	} else {
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git", t.user, t.repo)
	}
	cloneCmd := []string{"clone"}
	volumeMount := t.sharedVolumeMount()
	if t.branch != "" {
		cloneCmd = append(cloneCmd, "-b", t.branch, cloneURL, volumeMount.MountPath)
	} else {
		cloneCmd = append(cloneCmd, cloneURL, volumeMount.MountPath)
	}
	return apiv1.Container{
		Name:         "kubetest-init-clone",
		Image:        gitImageName,
		Command:      []string{"git"},
		Args:         cloneCmd,
		Env:          []apiv1.EnvVar{{Name: githubTokenEnv, Value: t.token}},
		VolumeMounts: []apiv1.VolumeMount{volumeMount},
	}
}

func (t *TestJob) gitSwitchContainer() apiv1.Container {
	volumeMount := t.sharedVolumeMount()
	return apiv1.Container{
		Name:         "kubetest-init-switch",
		Image:        gitImageName,
		WorkingDir:   volumeMount.MountPath,
		Command:      []string{"git"},
		Args:         []string{"switch", "--detach", t.rev},
		VolumeMounts: []apiv1.VolumeMount{volumeMount},
	}
}

func (t *TestJob) initContainers() []apiv1.Container {
	if t.branch != "" {
		return []apiv1.Container{t.gitCloneContainer()}
	}
	return []apiv1.Container{
		t.gitCloneContainer(),
		t.gitSwitchContainer(),
	}
}

func (t *TestJob) DisablePrepareLog() {
	t.disabledPrepareLog = true
}

func (t *TestJob) DisableCommandLog() {
	t.disabledCommandLog = true
}

func (t *TestJob) SetWriter(w io.Writer) {
	t.w = w
}

func (t *TestJob) Run(ctx context.Context) error {
	volumeMount := t.sharedVolumeMount()
	job, err := kubejob.NewJobBuilder(t.clientset, t.namespace).
		BuildWithJob(&batchv1.Job{
			Spec: batchv1.JobSpec{
				Template: apiv1.PodTemplateSpec{
					Spec: apiv1.PodSpec{
						Volumes:        []apiv1.Volume{t.sharedVolume()},
						InitContainers: t.initContainers(),
						Containers: []apiv1.Container{
							{
								Image:        t.image,
								Command:      []string{t.cmd[0]},
								Args:         t.cmd[1:],
								WorkingDir:   volumeMount.MountPath,
								VolumeMounts: []apiv1.VolumeMount{volumeMount},
							},
						},
					},
				},
			},
		})
	if err != nil {
		return xerrors.Errorf("failed to build testjob: %w", err)
	}
	if t.w != nil {
		job.SetWriter(t.w)
	}
	if t.disabledPrepareLog {
		job.DisableInitContainerLog()
	}
	if t.disabledCommandLog {
		job.DisableCommandLog()
	}
	if err := job.Run(ctx); err != nil {
		return xerrors.Errorf("failed to run testjob: %w", err)
	}
	return nil
}
