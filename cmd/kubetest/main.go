package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/kubetest"
	"github.com/jessevdk/go-flags"
	"golang.org/x/xerrors"
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
	Branch          string `description:"specify branch name" short:"b" long:"branch"`
	Revision        string `description:"specify revision ( commit hash )" long:"rev"`
	User            string `description:"specify user ( organization ) name" long:"user"`
	Repo            string `description:"specify repository name" long:"repo"`
	Token           string `description:"specify github auth token" long:"token"`
	TokenFromSecret string `description:"specify github auth token from secret resource. specify ( name.key ) style" long:"token-from-secret"`
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
	if opt.Image == "" {
		return xerrors.Errorf("image must be specified")
	}
	if opt.Repo == "" {
		return xerrors.Errorf("repo must be specified")
	}
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
	builder := kubetest.NewTestJobBuilder(clientset, opt.Namespace).
		SetUser(opt.User).
		SetRepo(opt.Repo).
		SetBranch(opt.Branch).
		SetImage(opt.Image).
		SetRev(opt.Revision).
		SetToken(opt.Token).
		SetCommand(args)
	if opt.TokenFromSecret != "" {
		splitted := strings.Split(opt.TokenFromSecret, ".")
		if len(splitted) != 2 {
			return xerrors.Errorf("invalid --token-from-secret parameter")
		}
		name := splitted[0]
		key := splitted[1]
		builder = builder.SetTokenFromSecret(name, key)
	}
	job, err := builder.Build()
	if err != nil {
		return xerrors.Errorf("failed to build testjob: %w", err)
	}
	if err := job.Run(context.Background()); err != nil {
		return xerrors.Errorf("failed to run testjob: %w", err)
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
		fmt.Printf("%+v", err)
	}
}
