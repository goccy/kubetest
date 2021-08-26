package kubetest

import (
	"testing"
)

func TestMask(t *testing.T) {
	msg := NewMaskedMessage("", []string{"awesome-secret", "awesome-password"})
	msg.AddMessage("aaaaa awesome-secret bbbb")
	msg.AddMessage("ccccc awesome-password dddd")

	expected := "aaaaa ************** bbbbccccc **************** dddd"
	actual := msg.String()

	if expected != actual {
		t.Fatalf("failed to mask: %s", actual)
	}
}
