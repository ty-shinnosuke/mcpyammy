package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

const (
	CommandApply  = "apply"
	CommandImport = "import"

	MaxYAMLSize  = 1024 * 1024 // 1MB
	MaxNestLevel = 50

	SecureFileMode = 0600
	DirectoryMode  = 0755
)

var (
	osExit           func(int)
	runTUIFunc       func()
	applyConfigFunc  func(string)
	importConfigFunc func(string)
)

type OrderedServer struct {
	Name    string                 `yaml:"name"`
	Command string                 `yaml:"command,omitempty"`
	Args    []string               `yaml:"args,omitempty"`
	Env     map[string]string      `yaml:"env,omitempty"`
	Extra   map[string]interface{} `yaml:",inline"`
}

func init() {
	osExit = os.Exit
	runTUIFunc = runTUI
	applyConfigFunc = applyConfig
	importConfigFunc = importConfig
}

func main() {
	if len(os.Args) == 1 {
		runTUIFunc()
		return
	}

	command := os.Args[1]

	if len(os.Args) < 3 {
		fmt.Printf("Usage: mcp-setup %s <yaml-file>\n", command)
		osExit(1)
	}

	runner := &CLICommandRunner{}
	runner.runCommand(command, os.Args[2])
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  mcp-setup apply <yaml-file>   Apply configuration from YAML file")
	fmt.Println("  mcp-setup import <yaml-file>  Import existing mcp.json files to YAML format")
}

func parseYAMLSafely(yamlData []byte, maxSize int64, target interface{}) error {
	if int64(len(yamlData)) > maxSize {
		return fmt.Errorf("YAMLファイルサイズが上限(%dKB)を超えています: %dKB",
			maxSize/1024, int64(len(yamlData))/1024)
	}

	yamlStr := string(yamlData)
	maxNestLevel := MaxNestLevel
	currentLevel := 0
	maxDetected := 0

	for _, char := range yamlStr {
		switch char {
		case '{', '[':
			currentLevel++
			if currentLevel > maxDetected {
				maxDetected = currentLevel
			}
		case '}', ']':
			currentLevel--
		}

		if maxDetected > maxNestLevel {
			return fmt.Errorf("YAML構造が深すぎます（最大%d階層）: %d階層検出",
				maxNestLevel, maxDetected)
		}
	}

	if err := yaml.Unmarshal(yamlData, target); err != nil {
		return fmt.Errorf("YAML解析エラー: %v", err)
	}
	return nil
}
