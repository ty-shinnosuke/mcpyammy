package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BaseProcessor struct{}

func (p *BaseProcessor) getHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (p *BaseProcessor) loadAndValidateYAML(yamlFile string) (map[string]interface{}, error) {
	return loadAndValidateYAML(yamlFile)
}

func (p *BaseProcessor) processClients(yamlContent map[string]interface{}, processFunc func(string, map[string]interface{}, string) error) (int, error) {
	homeDir, err := p.getHomeDir()
	if err != nil {
		return 0, fmt.Errorf("error getting home directory: %v", err)
	}

	clients := yamlContent["clients"].(map[string]interface{})
	processedCount := 0

	for clientName, clientConfig := range clients {
		config, ok := clientConfig.(map[string]interface{})
		if !ok {
			fmt.Printf("Invalid configuration for client '%s'\n", clientName)
			continue
		}

		if err := processFunc(clientName, config, homeDir); err != nil {
			fmt.Printf("Processing failed for client '%s': %v\n", clientName, err)
			continue
		}
		processedCount++
	}

	return processedCount, nil
}

func processClientConfig(clientName string, clientConfig map[string]interface{}, homeDir string) error {
	pathStr, ok := clientConfig["path"].(string)
	if !ok {
		return fmt.Errorf("クライアント'%s'にパスが指定されていません", clientName)
	}
	validatedPath, err := validateSafePath(pathStr, homeDir)
	if err != nil {
		return fmt.Errorf("セキュリティリスク検出（クライアント'%s'): %v", clientName, err)
	}
	servers := extractClientServers(clientConfig)
	if len(servers) == 0 {
		return fmt.Errorf("クライアント'%s'にサーバー設定が見つかりません", clientName)
	}

	existing := make(map[string]interface{})
	if data, err := os.ReadFile(validatedPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("JSON解析エラー（クライアント'%s'): %v", clientName, err)
		}
	}

	if existing["mcpServers"] == nil {
		existing["mcpServers"] = make(map[string]interface{})
	}
	mcpServers := existing["mcpServers"].(map[string]interface{})
	for name, server := range servers {
		mcpServers[name] = server
	}

	if err := os.MkdirAll(filepath.Dir(validatedPath), DirectoryMode); err != nil {
		return fmt.Errorf("ディレクトリ作成エラー（クライアント'%s'): %v", clientName, err)
	}

	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON生成エラー（クライアント'%s'): %v", clientName, err)
	}

	if err := os.WriteFile(validatedPath, output, SecureFileMode); err != nil {
		return fmt.Errorf("ファイル書き込みエラー（クライアント'%s'): %v", clientName, err)
	}

	fmt.Printf("✓ Updated %s: %s\n", clientName, validatedPath)
	return nil
}

func processSingleClientImportWithOutput(clientName string, config map[string]interface{}, homeDir string, outputClients map[string]interface{}) error {
	pathStr, ok := config["path"].(string)
	if !ok {
		fmt.Printf("No 'path' specified for client '%s', skipping...\n", clientName)
		return fmt.Errorf("no path specified")
	}

	originalPath := pathStr
	validatedPath, err := validateSafePath(pathStr, homeDir)
	if err != nil {
		fmt.Printf("Security risk detected for client '%s': %v\n", clientName, err)
		outputClients[clientName] = map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
		return err
	}

	pathStr = validatedPath
	jsonData, err := os.ReadFile(pathStr)
	if err != nil {
		outputClients[clientName] = map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
		fmt.Printf("- %s: file not found (path preserved)\n", clientName)
		return err
	}

	var jsonContent map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonContent); err != nil {
		outputClients[clientName] = map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
		fmt.Printf("- %s: parse error (path preserved)\n", clientName)
		return err
	}

	mcpServers, ok := jsonContent["mcpServers"].(map[string]interface{})
	if !ok || len(mcpServers) == 0 {
		outputClients[clientName] = map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
		fmt.Printf("- %s: no mcpServers (path preserved)\n", clientName)
		return fmt.Errorf("no mcpServers found")
	}

	outputClients[clientName] = map[string]interface{}{
		"path":    originalPath,
		"servers": convertMcpServersToYaml(mcpServers),
	}
	fmt.Printf("✓ Imported %s from %s\n", clientName, pathStr)
	return nil
}
