package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func AppHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, appDirName), nil
}

func GlobalRulesDir() (string, error) {
	appHome, err := AppHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appHome, "rules"), nil
}

func GlobalWorkflowsDir() (string, error) {
	appHome, err := AppHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appHome, "workflows"), nil
}

func WorkspaceRulesDir(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".agents", "rules")
}

func WorkspaceWorkflowsDir(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".agents", "workflows")
}
