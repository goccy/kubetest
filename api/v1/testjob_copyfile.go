package v1

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/kubejob"
	"golang.org/x/xerrors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

func (r *TestJobRunner) copyFile(executor *kubejob.JobExecutor, src, outputDir string) (e error) {
	pod := executor.Pod
	restClient := r.clientSet.CoreV1().RESTClient()
	req := restClient.Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Container: executor.Container.Name,
			Command:   []string{"tar", "cf", "-", src},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	url := req.URL()
	exec, err := remotecommand.NewSPDYExecutor(r.config, "POST", url)
	if err != nil {
		return xerrors.Errorf("failed to create spdy executor: %w", err)
	}
	reader, writer := io.Pipe()
	go func() {
		e = exec.Stream(remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: writer,
			Stderr: ioutil.Discard,
			Tty:    false,
		})
		writer.Close()
	}()
	prefix := getPrefix(src)
	prefix = path.Clean(prefix)
	prefix = stripPathShortcuts(prefix)
	return r.untarAll(src, reader, outputDir, prefix)
}

func (r *TestJobRunner) untarAll(src string, reader io.Reader, destDir, prefix string) error {
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		// All the files will start with the prefix, which is the directory where
		// they were located on the pod, we need to strip down that prefix, but
		// if the prefix is missing it means the tar was tempered with.
		// For the case where prefix is empty we need to ensure that the path
		// is not absolute, which also indicates the tar file was tempered with.
		if !strings.HasPrefix(header.Name, prefix) {
			return xerrors.Errorf("tar contents corrupted")
		}
		// basic file information
		mode := header.FileInfo().Mode()
		destFileName := filepath.Join(destDir, filepath.Base(src))
		if header.FileInfo().IsDir() {
			continue
		}

		if mode&os.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "warning: skipping symlink: %q -> %q\n", destFileName, header.Linkname)
			continue
		}
		outFile, err := os.Create(destFileName)
		if err != nil {
			return xerrors.Errorf("failed to create dst file: %w", err)
		}
		if _, err := io.Copy(outFile, tarReader); err != nil {
			return xerrors.Errorf("failed to copy: %w", err)
		}
		if err := outFile.Close(); err != nil {
			return xerrors.Errorf("failed to close file: %w", err)
		}
	}

	return nil
}

func stripPathShortcuts(p string) string {
	newPath := path.Clean(p)
	trimmed := strings.TrimPrefix(newPath, "../")

	for trimmed != newPath {
		newPath = trimmed
		trimmed = strings.TrimPrefix(newPath, "../")
	}

	// trim leftover {".", ".."}
	if newPath == "." || newPath == ".." {
		newPath = ""
	}

	if len(newPath) > 0 && string(newPath[0]) == "/" {
		return newPath[1:]
	}

	return newPath
}

func getPrefix(file string) string {
	// tar strips the leading '/' if it's there, so we will too
	return strings.TrimLeft(file, "/")
}
