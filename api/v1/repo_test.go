package v1

import (
	"context"
	"os"
	"testing"
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
}
