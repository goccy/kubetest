package v1

import (
	"context"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v29/github"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	tokenSecretName = "kubetest-git-token-"
	tokenSecretKey  = "kubetest-git-token-secret"
)

type tokenClient struct {
	clientSet *kubernetes.Clientset
	namespace string
}

func (t *TestJobToken) canUseSecretDirectly() bool {
	return t.GitHubToken != nil || t.Token != nil
}

func (t *TestJobToken) getTokenSecret() *corev1.SecretKeySelector {
	if t.GitHubToken != nil {
		return t.GitHubToken
	}
	if t.Token != nil {
		return t.Token
	}
	return nil
}

func boolptr(v bool) *bool {
	return &v
}

func (t *TestJobToken) createSecret(ctx context.Context, cli *tokenClient, token string) (*corev1.SecretKeySelector, error) {
	secret, err := cli.clientSet.CoreV1().
		Secrets(cli.namespace).
		Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: tokenSecretName,
			},
			Immutable: boolptr(true),
			Data: map[string][]byte{
				tokenSecretKey: []byte(token),
			},
		}, metav1.CreateOptions{})
	if err != nil {
		return nil, xerrors.Errorf("failed to create secret for token: %w", err)
	}
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name},
		Key:                  tokenSecretKey,
	}, nil
}

func (t *TestJobToken) deleteSecretBySelector(ctx context.Context, cli *tokenClient, secret *corev1.SecretKeySelector) error {
	if err := cli.clientSet.CoreV1().
		Secrets(cli.namespace).
		Delete(ctx, secret.Name, metav1.DeleteOptions{}); err != nil {
		return xerrors.Errorf("failed to delete secret for token: %w", err)
	}
	return nil
}

func (t *TestJobToken) AccessToken(ctx context.Context, cli *tokenClient) (string, error) {
	if t.GitHubApp != nil {
		tk, err := t.tokenFromGitHubApp(ctx, cli, t.GitHubApp)
		if err != nil {
			return "", xerrors.Errorf("failed to get token from github app settings: %w", err)
		}
		return tk, nil
	}
	if t.GitHubToken != nil {
		tk, err := t.tokenFromSecretRef(ctx, cli, t.GitHubToken)
		if err != nil {
			return "", xerrors.Errorf("failed to get token from github token settings: %w", err)
		}
		return tk, nil
	}
	if t.Token != nil {
		tk, err := t.tokenFromSecretRef(ctx, cli, t.Token)
		if err != nil {
			return "", xerrors.Errorf("failed to get token from token settings: %w", err)
		}
		return tk, nil
	}
	return "", nil
}

func (t *TestJobToken) tokenFromSecretRef(ctx context.Context, cli *tokenClient, param *corev1.SecretKeySelector) (string, error) {
	secret, err := cli.clientSet.CoreV1().
		Secrets(cli.namespace).
		Get(ctx, param.Name, metav1.GetOptions{})
	if err != nil {
		return "", xerrors.Errorf("failed to read secret for token by %s: %w", param.Name, err)
	}
	data, exists := secret.Data[param.Key]
	if !exists {
		return "", xerrors.Errorf("failed to find token data: %s", param.Key)
	}
	return strings.TrimSpace(string(data)), nil
}

func (t *TestJobToken) tokenFromGitHubApp(ctx context.Context, cli *tokenClient, param *GitHubAppTokenSpec) (string, error) {
	if param.AppID == 0 {
		return "", xerrors.Errorf("invalid param. appId is required to get token by github app settings")
	}
	if param.KeyFile == nil {
		return "", xerrors.Errorf("invalid param. keyFile is required to get token by github app settings")
	}
	if param.Organization == "" && param.InstallationID == 0 {
		return "", xerrors.Errorf("invalid param. organization or installationId is required to get token by github app settings")
	}
	privateKey, err := cli.clientSet.CoreV1().
		Secrets(cli.namespace).
		Get(ctx, param.KeyFile.Name, metav1.GetOptions{})
	if err != nil {
		return "", xerrors.Errorf("failed to read private key from secret %s: %w", param.KeyFile.Name, err)
	}
	privateKeyData, exists := privateKey.Data[param.KeyFile.Key]
	if !exists {
		return "", xerrors.Errorf("failed to find private key data: %s", param.KeyFile.Key)
	}
	token, err := t.tokenFromGitHubAppWithParam(ctx, param.AppID, param.InstallationID, param.Organization, privateKeyData)
	if err != nil {
		return "", xerrors.Errorf("failed to get token from github app params: %w", err)
	}
	return token, nil
}

func (t *TestJobToken) tokenFromGitHubAppWithParam(ctx context.Context, appID, installationID int64, org string, privateKey []byte) (string, error) {
	appsTransport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privateKey)
	if err != nil {
		return "", xerrors.Errorf("failed to initialize apps transport from %d: %w", appID, err)
	}
	githubClient := github.NewClient(&http.Client{Transport: appsTransport})
	if installationID == 0 {
		id, err := t.getInstallationID(ctx, githubClient, org)
		if err != nil {
			return "", xerrors.Errorf("failed to get installation id by %s: %w", org, err)
		}
		installationID = id
	}
	token, _, err := githubClient.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		return "", xerrors.Errorf("failed to create installation token: %w", err)
	}
	return token.GetToken(), nil
}

func (t *TestJobToken) getInstallationID(ctx context.Context, githubClient *github.Client, org string) (int64, error) {
	opt := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}
	for {
		ins, resp, err := githubClient.Apps.ListInstallations(ctx, opt)
		if err != nil {
			return 0, xerrors.Errorf("failed to fetch installations: %w", err)
		}
		for _, in := range ins {
			if org == in.GetAccount().GetLogin() {
				return in.GetID(), nil
			}
		}
		if resp.LastPage == 0 || opt.Page == resp.LastPage {
			return 0, xerrors.Errorf("failed to find %s in installations", org)
		}
		opt.Page++
	}
}
