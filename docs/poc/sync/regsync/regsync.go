package regsync

import (
	"fmt"
	"os/exec"
)

const (
	regsyncBin = "../../../bin/regsync-0.9.0"
	regsyncConfig = "regsync/regsync-config.yml"
)

func Sync(_, _, _, _ string) error {
	// regsync is a program not a library, so we need to run it from binary
	syncCmd := exec.Command(regsyncBin, "-c", regsyncConfig, "once")
	output, err := syncCmd.CombinedOutput()
	fmt.Printf("Regsync output:\n%s\n", output)
	if err != nil {
		return fmt.Errorf("failed to run regsync: %w", err)
	}

	return nil
}
