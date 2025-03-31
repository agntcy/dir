package hub

import (
	"os"
	"testing"
)

func TestHub(t *testing.T) {
	cmd := NewHubCommand()

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs([]string{"login"})
	cmd.Execute()
}
