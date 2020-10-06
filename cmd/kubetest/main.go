package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"github.com/jessevdk/go-flags"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
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
	List                string `description:"specify command for listing test" long:"list"`
	ListDelimiter       string `description:"specify delimiter for list command" long:"list-delimiter"`
	Pattern             string `description:"specify test name patter" long:"pattern"`
	Retest              *bool  `description:"specify enabled retest if exists failed tests" long:"retest"`
	RetestDelimiter     string `description:"specify delimiter for failed tests at retest command" long:"retest-delimiter"`

	File     string            `description:"specify yaml file path" short:"f" long:"file"`
	Template map[string]string `description:"specify template parameter for file specified with --file option" long:"template"`
}

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
	if opt.ListDelimiter != "" {
		return true
	}
	if opt.Pattern != "" {
		return true
	}
	if opt.Retest != nil {
		return true
	}
	if opt.RetestDelimiter != "" {
		return true
	}
	return false
}

func validateDistributedTestParam(job kubetestv1.TestJob) error {
	if job.Spec.DistributedTest.MaxContainersPerPod == 0 {
		return xerrors.New("the required flag '--max-containers-per-pod' was not specified")
	}
	if job.Spec.DistributedTest.ListCommand == "" {
		return xerrors.New("the required flag '--list' was not specified")
	}
	return nil
}

func validateTestJobParam(job kubetestv1.TestJob) error {
	if job.Spec.Image == "" {
		return xerrors.New("the required flag '--image' was not specified")
	}
	if job.Spec.Repo == "" {
		return xerrors.New("the required flag '--repo' was not specified")
	}
	if job.Spec.Command == "" {
		return xerrors.New("command is required. please speficy after '--' section")
	}
	return nil
}

func _main(args []string, opt option) error {
	cfg, err := loadConfig(opt)
	if err != nil {
		return xerrors.Errorf("failed to load config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return xerrors.Errorf("failed to create clientset: %w", err)
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
	if job.ObjectMeta.Namespace == "" {
		job.ObjectMeta.Namespace = opt.Namespace
	}
	if opt.Image != "" {
		job.Spec.Image = opt.Image
	}
	if opt.Repo != "" {
		job.Spec.Repo = opt.Repo
	}
	if opt.Branch != "" {
		job.Spec.Branch = opt.Branch
	}
	if opt.Revision != "" {
		job.Spec.Rev = opt.Revision
	}
	if len(args) > 0 {
		job.Spec.Command = kubetestv1.Command(strings.Join(args, " "))
	}
	if opt.TokenFromSecret != "" {
		splitted := strings.Split(opt.TokenFromSecret, ".")
		if len(splitted) != 2 {
			return xerrors.Errorf("invalid --token-from-secret parameter")
		}
		name := splitted[0]
		key := splitted[1]
		job.Spec.Token = &kubetestv1.TestJobToken{
			SecretKeyRef: kubetestv1.TestJobSecretKeyRef{
				Name: name,
				Key:  key,
			},
		}
	}
	if opt.ImagePullSecret != "" {
		job.Spec.ImagePullSecrets = append(job.Spec.ImagePullSecrets, corev1.LocalObjectReference{
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
			job.Spec.DistributedTest.ListCommand = kubetestv1.Command(opt.List)
		}
		if opt.ListDelimiter != "" {
			job.Spec.DistributedTest.ListDelimiter = opt.ListDelimiter
		}
		if opt.Pattern != "" {
			job.Spec.DistributedTest.Pattern = opt.Pattern
		}
		if opt.Retest != nil {
			job.Spec.DistributedTest.Retest = *opt.Retest
		}
		if opt.RetestDelimiter != "" {
			job.Spec.DistributedTest.RetestDelimiter = opt.RetestDelimiter
		}
		if err := validateDistributedTestParam(job); err != nil {
			return err
		}
	}
	if err := validateTestJobParam(job); err != nil {
		return err
	}
	if err := kubetestv1.NewTestJobRunner(clientset).Run(context.Background(), job); err != nil {
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
		os.Exit(1)
	}
}
