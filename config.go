package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

func applyConfig(yamlFile string) {
	processor := &BaseProcessor{}

	yamlContent, err := processor.loadAndValidateYAML(yamlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	processedCount, err := processor.processClients(yamlContent, processClientConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if processedCount > 0 {
		fmt.Printf("\nSuccessfully processed %d client(s)\n", processedCount)
	} else {
		fmt.Println("\nNo clients were processed")
	}
}

func importConfig(yamlFile string) {
	processor := &BaseProcessor{}

	yamlContent, err := processor.loadAndValidateYAML(yamlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	homeDir, err := processor.getHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	clients := yamlContent["clients"].(map[string]interface{})
	outputYaml := map[string]interface{}{
		"clients": make(map[string]interface{}),
	}
	outputClients := outputYaml["clients"].(map[string]interface{})
	importedCount := 0

	for clientName, clientConfig := range clients {
		config, ok := clientConfig.(map[string]interface{})
		if !ok {
			fmt.Printf("Invalid configuration for client '%s'\n", clientName)
			continue
		}

		if err := processSingleClientImportWithOutput(clientName, config, homeDir, outputClients); err != nil {
			continue
		}
		importedCount++
	}

	if importedCount == 0 {
		fmt.Println("No configurations were imported")
		os.Exit(1)
	}

	yamlBytes, err := yaml.Marshal(outputYaml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n--- Imported YAML configuration ---\n")
	fmt.Println(string(yamlBytes))
}

func loadAndValidateYAML(yamlFile string) (map[string]interface{}, error) {
	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("YAMLファイル読み込みエラー: %v", err)
	}
	var yamlContent map[string]interface{}
	if err := parseYAMLSafely(yamlData, MaxYAMLSize, &yamlContent); err != nil {
		return nil, fmt.Errorf("YAML検証エラー: %v", err)
	}
	if _, ok := yamlContent["clients"].(map[string]interface{}); !ok {
		return nil, fmt.Errorf("YAMLにclientsセクションが見つかりません")
	}
	return yamlContent, nil
}

func validateSafePath(pathStr, homeDir string) (string, error) {
	if pathStr == "" {
		return "", fmt.Errorf("パスが指定されていません")
	}

	cleanPath := filepath.Clean(pathStr)

	var resolvedPath string
	if after, found := strings.CutPrefix(cleanPath, "~"); found {
		resolvedPath = filepath.Join(homeDir, after)
	} else if !filepath.IsAbs(cleanPath) {
		resolvedPath = filepath.Join(homeDir, cleanPath)
	} else {
		resolvedPath = cleanPath
	}
	relPath, err := filepath.Rel(homeDir, resolvedPath)
	if err != nil {
		return "", fmt.Errorf("パスの検証に失敗しました: %v", err)
	}
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("ホームディレクトリ外へのアクセスは許可されていません: %s", pathStr)
	}
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("絶対パスの取得に失敗しました: %v", err)
	}
	return absPath, nil
}
