package v1

import (
	"testing"
)

func TestMask(t *testing.T) {
	msg := newMaskedMessage("", []string{"awesome-secret", "awesome-password"})
	msg.addMessage("aaaaa awesome-secret bbbb")
	msg.addMessage("ccccc awesome-password dddd")

	expected := "aaaaa ************** bbbbccccc **************** dddd"
	actual := msg.String()

	if expected != actual {
		t.Fatalf("failed to mask: %s", actual)
	}
}
