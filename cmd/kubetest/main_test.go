package main

import (
	"io/ioutil"
	"os"
	"testing"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"github.com/jessevdk/go-flags"
)

func TestHelpOpt(t *testing.T) {
	os.Args = []string{
		"kubetest",
		"--help",
	}
	if _, _, err := parseOpt(); err != nil {
		flagsErr, ok := err.(*flags.Error)
		if !ok {
			t.Fatalf("unknown error instance: %T", err)
		}
		if flagsErr.Type != flags.ErrHelp {
			t.Fatal(err)
		}
	}
}

func TestListOpt(t *testing.T) {
	t.Run("invalid list", func(t *testing.T) {
		t.Run("empty list", func(t *testing.T) {
			tmpfile, err := ioutil.TempFile("", "kubetest")
			if err != nil {
				t.Fatal(err)
			}
			os.Args = []string{
				"kubetest",
				"--list",
				tmpfile.Name(),
			}
			_, opt, err := parseOpt()
			if err != nil {
				t.Fatal(err)
			}
			job := kubetestv1.TestJob{
				Spec: kubetestv1.TestJobSpec{
					DistributedTest: &kubetestv1.DistributedTestSpec{},
				},
			}
			if err := assignListNames(&job, opt); err == nil {
				t.Fatal("expected error")
			}
		})
		t.Run("whitespace line list", func(t *testing.T) {
			tmpfile, err := ioutil.TempFile("", "kubetest")
			if err != nil {
				t.Fatal(err)
			}
			invalidList := `
    
    

`
			if err := ioutil.WriteFile(tmpfile.Name(), []byte(invalidList), 0644); err != nil {
				t.Fatal(err)
			}
			os.Args = []string{
				"kubetest",
				"--list",
				tmpfile.Name(),
			}
			_, opt, err := parseOpt()
			if err != nil {
				t.Fatal(err)
			}
			job := kubetestv1.TestJob{
				Spec: kubetestv1.TestJobSpec{
					DistributedTest: &kubetestv1.DistributedTestSpec{},
				},
			}
			if err := assignListNames(&job, opt); err == nil {
				t.Fatal("expected error")
			}
		})
	})
	t.Run("invalid opt", func(t *testing.T) {
		os.Args = []string{
			"kubetest",
			"--list",
		}
		if _, _, err := parseOpt(); err == nil {
			t.Fatal("expected error (expected argument for flag `--list')")
		}
	})
}
