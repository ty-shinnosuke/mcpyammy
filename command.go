package main

import "fmt"

// CommandRunner defines the interface for command execution
type CommandRunner interface {
	runCommand(command, yamlFile string)
}

// CLICommandRunner implements CommandRunner for CLI operations
type CLICommandRunner struct{}

// runCommand executes the specified command with the given YAML file
func (r *CLICommandRunner) runCommand(command, yamlFile string) {
	switch command {
	case CommandApply:
		applyConfigFunc(yamlFile)
	case CommandImport:
		importConfigFunc(yamlFile)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		osExit(1)
	}
}
