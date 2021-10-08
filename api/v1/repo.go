//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type RepositoryManager struct {
	repos        []RepositorySpec
	tokenMgr     *TokenManager
	clonedPaths  map[string]string
	archivePaths map[string]string
}

func NewRepositoryManager(repos []RepositorySpec, tokenMgr *TokenManager) *RepositoryManager {
	return &RepositoryManager{
		repos:        repos,
		tokenMgr:     tokenMgr,
		clonedPaths:  map[string]string{},
		archivePaths: map[string]string{},
	}
}

func (m *RepositoryManager) Cleanup() error {
	return nil
	errs := []string{}
	for name, clonedPath := range m.clonedPaths {
		if err := os.RemoveAll(clonedPath); err != nil {
			errs = append(errs, fmt.Sprintf("failed to remove %s repository directory: %s", name, err.Error()))
		}
	}
	for name, archivePath := range m.archivePaths {
		if err := os.RemoveAll(archivePath); err != nil {
			errs = append(errs, fmt.Sprintf("failed to remove %s repository archive directory: %s", name, err.Error()))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("kubetest: failed to cleanup %s", strings.Join(errs, ":"))
	}
	return nil
}

func (m *RepositoryManager) CloneAll(ctx context.Context) error {
	for _, repo := range m.repos {
		repoDir, err := os.MkdirTemp("", "repo")
		if err != nil {
			return fmt.Errorf("kubetest: failed to create temporary directory for repository: %w", err)
		}
		if err := m.clone(ctx, repoDir, repo.Value); err != nil {
			return err
		}
		repoArchiveDir, err := os.MkdirTemp("", "repo-archive")
		if err != nil {
			return fmt.Errorf("kubetest: failed to create temporary directory for repository archive: %w", err)
		}
		repoArchivePath := filepath.Join(repoArchiveDir, "repo.tar.gz")
		if err := m.archiveRepo(repoDir, repoArchivePath); err != nil {
			return err
		}
		m.archivePaths[repo.Name] = repoArchivePath
		m.clonedPaths[repo.Name] = repoDir
	}
	return nil
}

func (m *RepositoryManager) clone(ctx context.Context, clonedPath string, repo Repository) error {
	LoggerFromContext(ctx).Info("clone repository: %s", repo.URL)

	const (
		defaultBaseBranchName = "master"
		defaultRemoteName     = "origin"
	)

	if err := os.MkdirAll(clonedPath, 0755); err != nil {
		return fmt.Errorf("kubetest: failed to create directory %s for repository: %w", clonedPath, err)
	}
	var auth transport.AuthMethod
	if repo.Token != "" {
		token, err := m.tokenMgr.TokenByName(ctx, repo.Token)
		if err != nil {
			return err
		}
		auth = &http.BasicAuth{
			Username: "x-access-token",
			Password: token.Value,
		}
	}
	gitRepo, err := git.PlainCloneContext(ctx, clonedPath, false, &git.CloneOptions{
		URL:  repo.URL,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("kubetest: failed to clone repository: %w", err)
	}
	cfg, err := gitRepo.Config()
	if err != nil {
		return fmt.Errorf("kubetest: failed to get repository config: %w", err)
	}
	var remote string
	if len(cfg.Remotes) == 1 {
		for name := range cfg.Remotes {
			remote = name
			break
		}
	} else {
		remote = defaultRemoteName
	}
	var baseBranch string
	if cfg.Init.DefaultBranch != "" {
		baseBranch = cfg.Init.DefaultBranch
	} else if len(cfg.Branches) == 1 {
		for name := range cfg.Branches {
			baseBranch = name
			break
		}
	} else {
		baseBranch = defaultBaseBranchName
	}
	tree, err := gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("kubetest: failed to get worktree from repository: %w", err)
	}
	checkoutOpt := &git.CheckoutOptions{}
	switch {
	case repo.Branch != "":
		checkoutOpt.Branch = plumbing.NewRemoteReferenceName(remote, repo.Branch)
	case repo.Rev != "":
		checkoutOpt.Create = true
		checkoutOpt.Branch = plumbing.NewBranchReferenceName(repo.Rev)
		checkoutOpt.Hash = plumbing.NewHash(repo.Rev)
	}
	if err := checkoutOpt.Validate(); err != nil {
		return fmt.Errorf("kubetest: invalid checkout option: %w", err)
	}
	if err := tree.Checkout(checkoutOpt); err != nil {
		return fmt.Errorf("kubetest: failed to checkout: %w", err)
	}
	if repo.Merge != nil {
		if repo.Merge.Base != "" {
			baseBranch = repo.Merge.Base
		}
		var remoteName string
		for _, remote := range cfg.Remotes {
			remoteName = remote.Name
			break
		}
		// we'd like to use '--ff' strategy ( merge's default behavior ).
		// go-git doesn't support yet, so we use git client command.
		LoggerFromContext(ctx).Debug("merge base branch: git pull %s %s", remoteName, baseBranch)
		cmd := exec.Command("git", "pull", remoteName, baseBranch)
		cmd.Dir = clonedPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("kubetest: failed to merge base branch %s: %w", string(out), err)
		}
		LoggerFromContext(ctx).Info("%s", string(out))
	}
	return nil
}

func (m *RepositoryManager) archiveRepo(repoDir, archivePath string) error {
	dst, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("kubetest: failed to create archive file for repository: %w", err)
	}
	defer dst.Close()

	gzw, err := gzip.NewWriterLevel(dst, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("kubetest: failed to create gzip writer: %w", err)
	}
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("kubetest: failed to create archive file for repository: %w", err)
		}
		if info.IsDir() {
			return nil
		}
		name := path[len(repoDir)+1:]
		if err := tw.WriteHeader(&tar.Header{
			Name:    name,
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}); err != nil {
			return fmt.Errorf("kubetest: failed to write archive header to create archive file for repository: %w", err)
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("kubetest: failed to open local file to create archive file for repository: %w", err)
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("kubetest: failed to copy local file to archive file for repository: %w", err)
		}
		return nil
	})
}

func (m *RepositoryManager) ArchivePathByRepoName(name string) (string, error) {
	path, exists := m.archivePaths[name]
	if !exists {
		return "", fmt.Errorf("kubetest: repository name %s is undefined", name)
	}
	return path, nil
}
