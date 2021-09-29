package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"github.com/jessevdk/go-flags"
	"k8s.io/apimachinery/pkg/util/yaml"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type option struct {
	Namespace string            `description:"specify namespace" short:"n" long:"namespace" default:"default"`
	InCluster bool              `description:"specify whether in cluster" long:"in-cluster"`
	Config    string            `description:"specify local kubeconfig path. ( default: $HOME/.kube/config )" short:"c" long:"config"`
	List      string            `description:"specify path to get the list for test" long:"list"`
	LogLevel  string            `description:"specify log level (debug/info/warn/error)" long:"log-level"`
	DryRun    bool              `description:"specify dry run mode" long:"dry-run"`
	Template  map[string]string `description:"specify template parameter for testjob file" long:"template"`
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
			return nil, fmt.Errorf("kubetest: failed to load config in cluster: %w", err)
		}
		return cfg, nil
	}
	p := opt.Config
	if p == "" {
		p = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", p)
	if err != nil {
		return nil, fmt.Errorf("kubetest: failed to load config from %s: %w", p, err)
	}
	return cfg, nil
}

func assignStaticKeys(job *kubetestv1.TestJob, opt option) error {
	if opt.List == "" {
		return nil
	}

	list, err := os.ReadFile(opt.List)
	if err != nil {
		return fmt.Errorf("kubetest: failed to read list for test from %s: %w", opt.List, err)
	}
	staticKeys := []string{}
	for _, key := range strings.Split(string(list), "\n") {
		if strings.TrimSpace(key) == "" {
			continue
		}
		staticKeys = append(staticKeys, key)
	}
	if len(staticKeys) == 0 {
		return fmt.Errorf("kubetest: invalid list file. test list is empty")
	}
	return job.SetStaticStrategyKeys(staticKeys)
}

func _main(args []string, opt option) (*kubetestv1.Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("unspecified testjob file path")
	}
	path := args[0]
	cfg, err := loadConfig(opt)
	if err != nil {
		return nil, err
	}
	var job kubetestv1.TestJob
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("kubetest: failed to open %s: %w", path, err)
	}
	f, err := template.New("").Parse(string(file))
	if err != nil {
		return nil, fmt.Errorf("kubetest: failed to parse file as template %s: %w", string(file), err)
	}
	var b bytes.Buffer
	if err := f.Execute(&b, opt.Template); err != nil {
		return nil, fmt.Errorf("kubetest: failed to execute template %s: %w", string(file), err)
	}
	if err := yaml.NewYAMLOrJSONDecoder(&b, 1024).Decode(&job); err != nil {
		return nil, fmt.Errorf("kubetest: failed to decode YAML: %w", err)
	}
	if err := assignStaticKeys(&job, opt); err != nil {
		return nil, err
	}
	runMode := kubetestv1.RunModeKubernetes
	if opt.DryRun {
		runMode = kubetestv1.RunModeDryRun
	}
	runner := kubetestv1.NewRunner(cfg, runMode)
	switch opt.LogLevel {
	case "debug":
		runner.SetLogger(kubetestv1.NewLogger(os.Stdout, kubetestv1.LogLevelDebug))
	case "", "info":
		runner.SetLogger(kubetestv1.NewLogger(os.Stdout, kubetestv1.LogLevelInfo))
	case "warn":
		runner.SetLogger(kubetestv1.NewLogger(os.Stdout, kubetestv1.LogLevelWarn))
	case "error":
		runner.SetLogger(kubetestv1.NewLogger(os.Stdout, kubetestv1.LogLevelError))
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	canceledBySignal := false
	go func() {
		select {
		case s := <-interrupt:
			fmt.Fprintf(os.Stdout, "kubetest: receive %s. try to graceful stop\n", s)
			canceledBySignal = true
			cancel()
		}
	}()

	result, err := runner.Run(ctx, job)
	if err != nil {
		if canceledBySignal {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(ExitWithSignal)
		}
		return nil, err
	}
	return result, nil
}

func parseOpt() ([]string, option, error) {
	var opt option
	parser := flags.NewParser(&opt, flags.Default)
	args, err := parser.Parse()
	return args, opt, err
}

func main() {
	args, opt, err := parseOpt()
	if err != nil {
		flagsErr, ok := err.(*flags.Error)
		if !ok {
			fmt.Fprintf(os.Stderr, "kubetest: unknown parsed option error: %T %v\n", err, err)
			os.Exit(ExitWithOtherError)
		}
		if flagsErr.Type == flags.ErrHelp {
			return
		}
		os.Exit(ExitWithOtherError)
	}
	result, err := _main(args, opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitWithFatalError)
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(ExitWithFatalError)
	}
	fmt.Fprintln(os.Stdout, string(b))
	if result.Status == kubetestv1.ResultStatusFailure {
		os.Exit(ExitWithFailureTestJob)
	}
}
