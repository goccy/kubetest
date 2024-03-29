package v1

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestTokenManager(t *testing.T) {
	clientset, err := kubernetes.NewForConfig(getConfig())
	if err != nil {
		t.Fatal(err)
	}
	namespace := "default"
	gitHubToken := "ghp_foobar"
	tokenName := "test-github-token"
	if _, err := clientset.CoreV1().
		Secrets(namespace).
		Create(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: tokenName,
			},
			Data: map[string][]byte{
				"token": []byte(gitHubToken),
			},
		}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	defer func() {
		clientset.CoreV1().Secrets(namespace).
			Delete(context.Background(), tokenName, metav1.DeleteOptions{})
	}()

	cli := NewTokenClient(clientset, "default")
	mgr := NewTokenManager([]TokenSpec{
		{
			Name: "github-token",
			Value: TokenSource{
				GitHubToken: &GitHubTokenSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tokenName,
					},
					Key: "token",
				},
			},
		},
	}, cli)
	gotToken, err := mgr.TokenByName(WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelInfo)), "github-token")
	if err != nil {
		t.Fatal(err)
	}
	if gitHubToken != gotToken.Value {
		t.Fatalf("failed to get token. expected %s but got %s", gitHubToken, gotToken.Value)
	}
}

func TestTokenFromGitHubApp(t *testing.T) {
	var (
		appID = int64(134426)
		org   = "goccy"
	)
	// this private key is valid, but since I have not given any permission to the github app associated with this private-key,
	// there is nothing we can do with this key.
	// I use this only to verify the logic that creates access token using the information in the github app.
	// ( but there is nothing you can do with the access token you get )
	privateKeyPath := filepath.Join("..", "..", "testdata", "githubapp.private-key.pem")
	privateKey, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	token, err := new(TokenClient).tokenFromGitHubAppWithParam(
		context.Background(),
		appID,
		0,
		org,
		privateKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "ghs_") {
		t.Fatalf("failed to get valid token: %s", token)
	}
}

func TestTokenFromFilePath(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	tokenFilePath := filepath.Join(tmpdir, "token")
	tokenContent := []byte(`dummytoken`)
	if err := os.WriteFile(tokenFilePath, tokenContent, 0o600); err != nil {
		t.Fatal(err)
	}

	mgr := NewTokenManager([]TokenSpec{
		{
			Name: "filePathToken",
			Value: TokenSource{
				FilePath: &tokenFilePath,
			},
		},
	}, NewTokenClient(nil, "default"))
	ctx := WithLogger(context.Background(), NewLogger(os.Stdout, LogLevelInfo))
	token, err := mgr.TokenByName(ctx, "filePathToken")
	if err != nil {
		t.Fatal(err)
	}
	if token.Value != string(tokenContent) {
		t.Fatalf("failed to get token from file. expected %s but got %s", string(tokenContent), token.Value)
	}
}
