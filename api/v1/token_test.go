package v1

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

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
	token, err := new(TestJobToken).tokenFromGitHubAppWithParam(
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
