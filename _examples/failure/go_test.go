package test

import "testing"

func Test_A(t *testing.T) {
	t.Logf("Test_A")
}

func Test_B(t *testing.T) {
	t.Errorf("error Test_B")
}
