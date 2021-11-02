package v1

import (
	"context"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestRepositoryManager(t *testing.T) {
	t.Run("checkout branch", func(t *testing.T) {
		mgr := NewRepositoryManager([]RepositorySpec{
			{
				Name: "test",
				Value: Repository{
					URL:    "https://github.com/goccy/kubetest.git",
					Branch: "master",
					Merge:  &MergeSpec{},
				},
			},
		}, new(TokenManager))
		defer func() {
			if err := mgr.Cleanup(); err != nil {
				t.Fatal(err)
			}
		}()
		if err := mgr.CloneAll(WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelDebug))); err != nil {
			t.Fatal(err)
		}
		path, err := mgr.ArchivePathByRepoName("test")
		if err != nil {
			t.Fatal(err)
		}
		if path == "" {
			t.Fatal("failed to clone repository with branch")
		}
		t.Logf("checkout by branch. archive path: %s", path)
	})
	t.Run("checkout revision", func(t *testing.T) {
		mgr := NewRepositoryManager([]RepositorySpec{
			{
				Name: "test",
				Value: Repository{
					URL:   "https://github.com/goccy/kubetest.git",
					Rev:   "cc74ac0bc8c1e82ea362145e48a222388b018461", // initial commit revision
					Merge: &MergeSpec{},
				},
			},
		}, new(TokenManager))
		defer func() {
			if err := mgr.Cleanup(); err != nil {
				t.Fatal(err)
			}
		}()
		if err := mgr.CloneAll(WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelDebug))); err != nil {
			t.Fatal(err)
		}
		path, err := mgr.ArchivePathByRepoName("test")
		if err != nil {
			t.Fatal(err)
		}
		if path == "" {
			t.Fatal("failed to clone repository with revision")
		}
		t.Logf("checkout by revision. archive path: %s", path)
	})
	t.Run("reuse cloned directory", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "repo")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}()
		if _, err := git.PlainCloneContext(context.Background(), dir, false, &git.CloneOptions{
			URL: "https://github.com/goccy/kubetest.git",
		}); err != nil {
			t.Fatal(err)
		}
		mgr := NewRepositoryManager([]RepositorySpec{
			{
				Name:  "test",
				Value: Repository{ClonedPath: dir},
			},
		}, new(TokenManager))
		defer func() {
			if err := mgr.Cleanup(); err != nil {
				t.Fatal(err)
			}
		}()
		if err := mgr.CloneAll(WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelDebug))); err != nil {
			t.Fatal(err)
		}
		path, err := mgr.ArchivePathByRepoName("test")
		if err != nil {
			t.Fatal(err)
		}
		if path == "" {
			t.Fatal("failed to get archive path")
		}
		t.Logf("archive path: %s", path)
	})
}
