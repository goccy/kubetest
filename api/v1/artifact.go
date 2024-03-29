//go:build !ignore_autogenerated
// +build !ignore_autogenerated

package v1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type ArtifactManager struct {
	nameToLocalDirs  map[string]string
	nameToLocalFiles map[string]string
	exports          []ExportArtifact
}

func NewArtifactManager(exports []ExportArtifact) *ArtifactManager {
	return &ArtifactManager{
		nameToLocalDirs:  map[string]string{},
		nameToLocalFiles: map[string]string{},
		exports:          exports,
	}
}

func (m *ArtifactManager) AddArtifacts(artifacts []ArtifactSpec) error {
	for _, artifact := range artifacts {
		dir, err := os.MkdirTemp("", "artifact")
		if err != nil {
			return fmt.Errorf("kubetest: failed to create temporary directory for artifact: %w", err)
		}
		m.nameToLocalDirs[artifact.Name] = dir
		m.nameToLocalFiles[artifact.Name] = filepath.Base(artifact.Container.Path)
	}
	return nil
}

func (m *ArtifactManager) ExportPathByName(name string) (string, error) {
	dir, exists := m.nameToLocalDirs[name]
	if !exists {
		return "", fmt.Errorf("kubetest: failed to find src path to export artifact by %s", name)
	}
	return dir, nil
}

func (m *ArtifactManager) LocalPathByName(ctx context.Context, name string) (string, error) {
	dir, exists := m.nameToLocalDirs[name]
	if !exists {
		return "", fmt.Errorf("kubetest: failed to find local artifact directory by %s", name)
	}
	file, exists := m.nameToLocalFiles[name]
	if !exists {
		return "", fmt.Errorf("kubetest: failed to find local artitfact file by %s", name)
	}
	containerNames, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return "", fmt.Errorf("kubetest: couldn't find local path for artifact %s", name)
	}
	if len(containerNames) == 0 {
		return "", fmt.Errorf("kubetest: couldn't find local path for artifact %s", name)
	}
	if len(containerNames) > 1 {
		LoggerFromContext(ctx).Info(
			"multiple paths to artifact were found. As for the copy destination path, %s ~ %s directories are placed as an intermediate directory",
			filepath.Base(containerNames[0]),
			filepath.Base(containerNames[len(containerNames)-1]),
		)
		return dir, nil
	}
	containerName := filepath.Base(containerNames[0])
	return filepath.Join(dir, containerName, file), nil
}

func (m *ArtifactManager) LocalPathByNameAndContainerName(name, containerName string) (string, error) {
	dir, exists := m.nameToLocalDirs[name]
	if !exists {
		return "", fmt.Errorf("kubetest: failed to find local artifact directory by %s", name)
	}
	file, exists := m.nameToLocalFiles[name]
	if !exists {
		return "", fmt.Errorf("kubetest: failed to find local artifact file by %s", name)
	}
	return filepath.Join(dir, containerName, file), nil
}

func (m *ArtifactManager) ExportArtifacts(ctx context.Context) error {
	for _, export := range m.exports {
		LoggerFromContext(ctx).Info("export artifact %s", export.Name)
		src, err := m.ExportPathByName(export.Name)
		if err != nil {
			return fmt.Errorf("kubetest: failed to get src path to export artifact: %w", err)
		}
		dst := export.Path
		if err := os.MkdirAll(dst, 0755); err != nil {
			return fmt.Errorf("kubetest: failed to create %s directory for export artifact: %w", dst, err)
		}
		paths, err := filepath.Glob(filepath.Join(src, "*"))
		if err != nil {
			return fmt.Errorf("kubetest: failed to get src path to export artifact: %w", err)
		}
		for _, path := range paths {
			src := path
			dst := filepath.Join(dst, filepath.Base(path))
			LoggerFromContext(ctx).Debug(
				"export artifact: copy from %s to %s",
				src, dst,
			)
			if err := localCopy(src, dst); err != nil {
				return err
			}
		}
	}
	return nil
}
