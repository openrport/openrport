package chclient

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func ValidateScriptDir(scriptDir string) error {
	if strings.TrimSpace(scriptDir) == "" {
		return errors.New("script directory cannot be empty")
	}

	dirStat, err := os.Stat(scriptDir)

	if os.IsNotExist(err) {
		return fmt.Errorf("script directory %s does not exist", scriptDir)
	}
	if err != nil {
		return err
	}
	if !dirStat.IsDir() {
		return fmt.Errorf("script directory %s is not a valid directory", scriptDir)
	}

	err = ValidateScriptDirOS(dirStat, scriptDir)
	if err != nil {
		return err
	}

	return nil
}
