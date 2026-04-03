package base

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetProjectRootDir returns absolute path of the project root directory.
// It walks up from the current working directory until it finds a "go.mod" file,
// which is used as the pivot to detect the project root.
// This works regardless of which subdirectory the binary is executed from,
// as long as the project source tree is available on disk.
func GetProjectRootDir() (string, error) {
	const pivot = "go.mod"
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("os.Getwd: %w", err)
	}
	for range 32 {
		pivotFilePath := filepath.Join(workingDir, pivot)
		if _, err := os.Stat(pivotFilePath); err == nil {
			// found the pivot "go.mod" file
			return workingDir, nil
		}
		parent := filepath.Dir(workingDir)
		if parent == workingDir {
			return "", fmt.Errorf("reached filesystem root without finding %s", pivot)
		}
		workingDir = parent
	}
	return "", fmt.Errorf("exceeded max iterations looking for %s", pivot)
}
