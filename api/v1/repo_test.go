package v1

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/sosedoff/gitkit"
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
		spec := RepositorySpec{
			Name:  "test",
			Value: Repository{ClonedPath: dir},
		}
		if err := NewValidator().ValidateRepositorySpec(spec); err != nil {
			t.Fatal(err)
		}
		mgr := NewRepositoryManager([]RepositorySpec{spec}, new(TokenManager))
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
	t.Run("add a file that specified to be ignored on the base branch", func(t *testing.T) {
		addr, reposDir := runGitServer(t)

		// create test repository
		repoName := "test"
		fs := osfs.New(filepath.Join(reposDir, repoName))
		storage := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
		repo, err := git.Init(storage, fs)
		if err != nil {
			t.Fatal(err)
		}
		w, err := repo.Worktree()
		if err != nil {
			t.Fatal(err)
		}

		// create .gitignore
		f, err := fs.Create(".gitignore")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write([]byte("*.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Add(".gitignore"); err != nil {
			t.Fatal(err)
		}

		// commit1
		commit1, err := w.Commit("commit1", &git.CommitOptions{})
		if err != nil {
			t.Fatal(err)
		}

		// update .gitignore
		f, err = fs.OpenFile(".gitignore", os.O_RDWR|os.O_APPEND, 0o666)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("\n!test.txt")); err != nil {
			t.Fatal(err)
		}

		// create test.txt
		f, err = fs.Create("test.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write([]byte("test")); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Add("test.txt"); err != nil {
			t.Fatal(err)
		}

		// commit2
		commit2, err := w.Commit("commit2", &git.CommitOptions{All: true})
		if err != nil {
			t.Fatal(err)
		}

		// set refs/heads/master => commit1
		master := plumbing.NewHashReference("refs/heads/master", commit1)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Storer.SetReference(master); err != nil {
			t.Fatal(err)
		}

		// set refs/heads/feature => commit2
		feature := plumbing.NewHashReference("refs/heads/feature", commit2)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Storer.SetReference(feature); err != nil {
			t.Fatal(err)
		}

		// clone by RepositoryManager
		repoDir := filepath.Join(t.TempDir(), repoName)
		spec := RepositorySpec{
			Name: repoName,
			Value: Repository{
				URL: fmt.Sprintf("http://%s/%s", addr, repoName),
				Rev: feature.Hash().String(),
				Merge: &MergeSpec{
					Base: "master",
				},
				ClonedPath: repoDir,
			},
		}
		if err := NewValidator().ValidateRepositorySpec(spec); err != nil {
			t.Fatal(err)
		}
		mgr := NewRepositoryManager([]RepositorySpec{spec}, new(TokenManager))
		t.Cleanup(func() {
			mgr.Cleanup()
		})
		if err := mgr.CloneAll(WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelDebug))); err != nil {
			t.Fatal(err)
		}

		repo, err = git.PlainOpen(repoDir)
		if err != nil {
			t.Fatal(err)
		}
		w, err = repo.Worktree()
		if err != nil {
			t.Fatal(err)
		}
		assertFile(t, w.Filesystem, ".gitignore", "*.txt\n!test.txt")
		assertFile(t, w.Filesystem, "test.txt", "test")
	})
}

func runGitServer(t *testing.T) (string, string) {
	t.Helper()

	tempDir := t.TempDir()
	reposDir := filepath.Join(tempDir, "repos")
	h := gitkit.New(gitkit.Config{
		Dir: reposDir,
	})
	if err := h.Setup(); err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{
		Handler: h,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = srv.Close()
	})

	// create test config
	configDir := filepath.Join(tempDir, "config")
	t.Setenv("XDG_CONFIG_HOME", configDir)
	gitConfigDir := filepath.Join(configDir, "git")
	if err := os.MkdirAll(gitConfigDir, 0o744); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(gitConfigDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(`[user]
	name = kubetest
	email = kubetest@example.com`); err != nil {
		t.Fatal(err)
	}

	return ln.Addr().String(), reposDir
}

func assertFile(t *testing.T, fs billy.Filesystem, path string, expect string) {
	t.Helper()

	f, err := fs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != expect {
		t.Errorf("%s: expect %q but got %q", path, expect, got)
	}
}
