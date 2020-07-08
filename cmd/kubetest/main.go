package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"github.com/jessevdk/go-flags"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type option struct {
	Namespace       string `description:"specify namespace" short:"n" long:"namespace" default:"default"`
	InCluster       bool   `description:"specify whether in cluster" long:"in-cluster"`
	Config          string `description:"specify local kubeconfig path. ( default: $HOME/.kube/config )" short:"c" long:"config"`
	Image           string `description:"specify container image" short:"i" long:"image" required:"true"`
	Repo            string `description:"specify repository name" long:"repo" required:"true"`
	Branch          string `description:"specify branch name" short:"b" long:"branch"`
	Revision        string `description:"specify revision ( commit hash )" long:"rev"`
	TokenFromSecret string `description:"specify github auth token from secret resource. specify ( name.key ) style" long:"token-from-secret"`
	ImagePullSecret string `description:"specify image pull secret name" long:"image-pull-secret"`
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

func _main(args []string, opt option) error {
	if opt.Branch == "" && opt.Revision == "" {
		return xerrors.Errorf("branch or rev must be specified")
	}
	if len(args) == 0 {
		return xerrors.Errorf("command is required. please speficy after '--' section")
	}

	cfg, err := loadConfig(opt)
	if err != nil {
		return xerrors.Errorf("failed to load config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return xerrors.Errorf("failed to create clientset: %w", err)
	}
	job := kubetestv1.TestJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opt.Namespace,
		},
		Spec: kubetestv1.TestJobSpec{
			Image:   opt.Image,
			Repo:    opt.Repo,
			Branch:  opt.Branch,
			Rev:     opt.Revision,
			Command: args,
		},
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
