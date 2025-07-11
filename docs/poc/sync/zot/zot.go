package zot

import (
	"fmt"
	"os/exec"
	"time"
)

func Sync(sourceRegistry, sourceRepo, targetRegistry, targetRepo string) error {
	newConfigFile := "./zot/zot-target-config-sync.json"
	configFile := "./zot-target-config.json"

	// Run command to move the new config file to the config file
	command := fmt.Sprintf("cp %s %s", newConfigFile, configFile)
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		return fmt.Errorf("failed to move config file: %w", err)
	}

	// Wait for zot to sync
	time.Sleep(1 * time.Minute)

	return nil
}
