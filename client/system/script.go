package system

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/random"
)

const DefaultFileMode = os.FileMode(0540)
const DefaultDirMode = os.FileMode(0700)

func CreateScriptFile(scriptDir, scriptContent string, interpreter Interpreter, encoder *encoding.Encoder) (filePath string, err error) {
	err = ValidateScriptDir(scriptDir)
	if err != nil {
		return "", err
	}

	scriptFileName, err := createScriptFileName(interpreter)
	if err != nil {
		return "", err
	}

	scriptFilePath := filepath.Join(scriptDir, scriptFileName)

	byteContent := []byte(scriptContent)
	if encoder != nil {
		byteContent, err = encoder.Bytes(byteContent)
		if err != nil {
			return "", err
		}
	}

	err = ioutil.WriteFile(scriptFilePath, byteContent, DefaultFileMode)
	if err != nil {
		return "", err
	}

	return scriptFilePath, nil
}

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

func createScriptFileName(interpreter Interpreter) (string, error) {
	scriptName, err := random.UUID4()
	if err != nil {
		return "", err
	}

	extension := getExtension(interpreter)

	return scriptName + extension, nil
}

func getExtension(interpreter Interpreter) string {
	if interpreter.InterpreterNameFromInput == chshare.Tacoscript {
		return ".yml"
	}

	return GetScriptExtensionOS(interpreter)
}

const shebangPrefix = "#!"

// HasShebangLine is just for making code more readable
func HasShebangLine(script string) bool {
	return strings.HasPrefix(script, shebangPrefix)
}
