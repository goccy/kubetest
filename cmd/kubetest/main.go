package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"github.com/jessevdk/go-flags"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type option struct {
	Namespace       string `description:"specify namespace" short:"n" long:"namespace" default:"default"`
	InCluster       bool   `description:"specify whether in cluster" long:"in-cluster"`
	Config          string `description:"specify local kubeconfig path. ( default: $HOME/.kube/config )" short:"c" long:"config"`
	Image           string `description:"specify container image" short:"i" long:"image"`
	Repo            string `description:"specify repository name" long:"repo"`
	Branch          string `description:"specify branch name" short:"b" long:"branch"`
	Revision        string `description:"specify revision ( commit hash )" long:"rev"`
	TokenFromSecret string `description:"specify github auth token from secret resource. specify ( name.key ) style" long:"token-from-secret"`
	ImagePullSecret string `description:"specify image pull secret name" long:"image-pull-secret"`

	// Distributed Testing Parameters
	MaxContainersPerPod int    `description:"specify max number of container per pod" long:"max-containers-per-pod"`
	List                string `description:"specify path to get the list for test" long:"list"`
	Retest              *bool  `description:"specify enabled retest if exists failed tests" long:"retest"`
	Verbose             bool   `description:"specify enabled debug log" short:"v" long:"versbose"`

	File     string            `description:"specify yaml file path" short:"f" long:"file"`
	Template map[string]string `description:"specify template parameter for file specified with --file option" long:"template"`
}

const (
	ExitSuccess            int = 0
	ExitWithFailureTestJob     = 1
	ExitWithOtherError         = 2
	ExitWithFatalError         = 3
	ExitWithSignal             = 4
)

func loadConfig(opt option) (*rest.Config, error) {
	if opt.InCluster {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, xerrors.Errorf("failed to load config in cluster: %w", err)
		}
		return cfg, nil
	}
	p := opt.Config
	if p == "" {
		p = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", p)
	if err != nil {
		return nil, xerrors.Errorf("failed to load config from %s: %w", p, err)
	}
	return cfg, nil
}

func hasDistributedParam(job kubetestv1.TestJob, opt option) bool {
	if job.Spec.DistributedTest != nil {
		return true
	}
	if opt.MaxContainersPerPod > 0 {
		return true
	}
	if opt.List != "" {
		return true
	}
	if opt.Retest != nil {
		return true
	}
	return false
}

func validateDistributedTestParam(job kubetestv1.TestJob) error {
	if job.Spec.DistributedTest.MaxContainersPerPod == 0 {
		return xerrors.New("the required flag '--max-containers-per-pod' was not specified")
	}
	if len(job.Spec.DistributedTest.List.Command) == 0 {
		return xerrors.New("the required flag '--list' was not specified")
	}
	return nil
}

func validateTestJobParam(job kubetestv1.TestJob) error {
	if job.Spec.Git.Checkout != nil && !(*job.Spec.Git.Checkout) {
		return nil
	}
	if job.Spec.Git.Repo == "" {
		return xerrors.New("the required flag '--repo' was not specified")
	}
	if job.Spec.Template.Spec.Containers[0].Image == "" {
		return xerrors.New("the required flag '--image' was not specified")
	}
	if len(job.Spec.Template.Spec.Containers[0].Command) == 0 {
		return xerrors.New("command is required. please speficy after '--' section")
	}
	return nil
}

func _main(args []string, opt option) error {
	cfg, err := loadConfig(opt)
	if err != nil {
		return xerrors.Errorf("failed to load config: %w", err)
	}
	var job kubetestv1.TestJob
	if opt.File != "" {
		file, err := ioutil.ReadFile(opt.File)
		if err != nil {
			return xerrors.Errorf("failed to open %s: %w", string(file), err)
		}
		f, err := template.New("").Parse(string(file))
		if err != nil {
			return xerrors.Errorf("failed to parse file as template %s: %w", string(file), err)
		}
		var b bytes.Buffer
		if err := f.Execute(&b, opt.Template); err != nil {
			return xerrors.Errorf("failed to execute template %s: %w", string(file), err)
		}
		if err := yaml.NewYAMLOrJSONDecoder(&b, 1024).Decode(&job); err != nil {
			return xerrors.Errorf("failed to decode YAML: %w", err)
		}
	}
	if len(job.Spec.Template.Spec.Containers) == 0 {
		job.Spec.Template.Spec.Containers = []corev1.Container{{}}
	}
	if job.ObjectMeta.Namespace == "" {
		job.ObjectMeta.Namespace = opt.Namespace
	}
	if opt.Image != "" {
		job.Spec.Template.Spec.Containers[0].Image = opt.Image
	}
	if opt.Repo != "" {
		job.Spec.Git.Repo = opt.Repo
	}
	if opt.Branch != "" {
		job.Spec.Git.Branch = opt.Branch
	}
	if opt.Revision != "" {
		job.Spec.Git.Rev = opt.Revision
	}
	if len(args) > 0 {
		job.Spec.Template.Spec.Containers[0].Command = []string{args[0]}
		if len(args) > 1 {
			job.Spec.Template.Spec.Containers[0].Args = args[1:]
		}
	}
	if opt.TokenFromSecret != "" {
		splitted := strings.Split(opt.TokenFromSecret, ".")
		if len(splitted) != 2 {
			return xerrors.Errorf("invalid --token-from-secret parameter")
		}
		name := splitted[0]
		key := splitted[1]
		job.Spec.Git.Token = &kubetestv1.TestJobToken{
			SecretKeyRef: kubetestv1.TestJobSecretKeyRef{
				Name: name,
				Key:  key,
			},
		}
	}
	if opt.ImagePullSecret != "" {
		job.Spec.Template.Spec.ImagePullSecrets = append(job.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: opt.ImagePullSecret,
		})
	}

	if hasDistributedParam(job, opt) {
		if job.Spec.DistributedTest == nil {
			job.Spec.DistributedTest = &kubetestv1.DistributedTestSpec{}
		}
		if opt.MaxContainersPerPod > 0 {
			job.Spec.DistributedTest.MaxContainersPerPod = opt.MaxContainersPerPod
		}
		if opt.List != "" {
			list, err := ioutil.ReadFile(opt.List)
			if err != nil {
				return xerrors.Errorf("failed to read list for test from %s: %w", opt.List, err)
			}
			testNames := strings.Split(string(list), "\n")
			job.Spec.DistributedTest.List.Names = testNames
		}
		if opt.Retest != nil {
			job.Spec.DistributedTest.Retest = *opt.Retest
		}
		if err := validateDistributedTestParam(job); err != nil {
			return err
		}
	}
	if err := validateTestJobParam(job); err != nil {
		return err
	}
	kubetest, err := kubetestv1.NewTestJobRunner(cfg)
	if err != nil {
		return err
	}
	if opt.Verbose {
		kubetest.EnableVerboseLog()
	}

	ctx, cancel := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	receiveSignal := false
	go func() {
		select {
		case s := <-interrupt:
			fmt.Printf("receive %s. try to graceful stop\n", s)
			receiveSignal = true
			cancel()
		}
	}()

	if err := kubetest.Run(ctx, job); err != nil {
		if receiveSignal {
			fmt.Println(err)
			os.Exit(ExitWithSignal)
		}
		return err
	}
	return nil
}

func main() {
	var opt option
	parser := flags.NewParser(&opt, flags.Default)
	args, err := parser.Parse()
	if err != nil {
		return
	}
	if err := _main(args, opt); err != nil {
		fmt.Println(err)
		if xerrors.Is(err, kubetestv1.ErrFailedTestJob) {
			os.Exit(ExitWithFailureTestJob)
		} else if xerrors.Is(err, kubetestv1.ErrFatal) {
			os.Exit(ExitWithFatalError)
		}
		os.Exit(ExitWithOtherError)
	}
}
