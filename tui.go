package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
)

const (
	DefaultViewportWidth      = 80
	DefaultViewportHeight     = 20
	ViewportHorizontalPadding = 2
	ViewportVerticalPadding   = 6
	TUIFileMode               = 0600
	TUIDirectoryMode          = 0755
)

const defaultYAML = `clients:
  amazonq:
    path: .aws/amazonq/mcp.json
    servers:
  gemini:
    path: .gemini/settings.json
    servers:
  claude:
    path: .claude.json
    servers:
`

type state int

const (
	stateMenu state = iota
	stateImport
	stateApply
	stateConfirm
	stateResult
)

type model struct {
	state       state
	list        list.Model
	viewport    viewport.Model
	yamlFile    string
	yamlContent string
	action      string
	result      string
	width       int
	height      int
	err         error
	yesNoIndex  int
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

func initialModel() model {
	items := []list.Item{
		item{title: "Import", desc: "Import existing mcp.json files to YAML"},
		item{title: "Apply", desc: "Apply YAML configuration to mcp.json files"},
		item{title: "Quit", desc: "Exit the program"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "MCP Setup"
	l.SetShowStatusBar(false)

	vp := viewport.New(DefaultViewportWidth, DefaultViewportHeight)

	return model{
		state:    stateMenu,
		list:     l,
		viewport: vp,
		yamlFile: "servers.yaml",
	}
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd {
	if _, err := os.Stat(m.yamlFile); os.IsNotExist(err) {
		_ = os.WriteFile(m.yamlFile, []byte(defaultYAML), TUIFileMode)
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			switch m.state {
			case stateMenu:
				return m, tea.Quit
			default:
				m.state = stateMenu
				return m, nil
			}
		case "left", "h":
			switch m.state {
			case stateConfirm:
				m.yesNoIndex = 0 // Yes
			}
		case "right", "l":
			switch m.state {
			case stateConfirm:
				m.yesNoIndex = 1 // No
			}
		case "enter":
			switch m.state {
			case stateMenu:
				selected := m.list.SelectedItem().(item).title
				switch selected {
				case "Import":
					m.action = CommandImport
					m.state = stateImport
					m.yesNoIndex = 0
					return m, m.runImport()
				case "Apply":
					m.action = CommandApply
					m.state = stateApply
					m.yesNoIndex = 0
					return m, m.runApplyPreview()
				case "Quit":
					return m, tea.Quit
				}
			case stateConfirm:
				if m.action == CommandApply && strings.Contains(m.viewport.View(), "No changes detected") {
					m.state = stateMenu
					return m, nil
				}
				if m.yesNoIndex == 0 {
					return m, m.executeAction()
				} else {
					m.state = stateMenu
				}
			case stateResult:
				m.state = stateMenu
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		m.viewport.Width = msg.Width - ViewportHorizontalPadding
		m.viewport.Height = msg.Height - ViewportVerticalPadding

	case importResult:
		m.yamlContent = string(msg)
		m.state = stateConfirm
		m.viewport.SetContent(m.yamlContent)

	case applyPreviewResult:
		m.result = string(msg)
		m.state = stateConfirm
		m.viewport.SetContent(m.result)

	case actionComplete:
		m.result = string(msg)
		m.state = stateResult
		m.viewport.SetContent(m.result)

	case errMsg:
		m.err = msg.err
		m.state = stateResult
		m.result = errorStyle.Render("Error: " + msg.err.Error())
		m.viewport.SetContent(m.result)
	}

	switch m.state {
	case stateMenu:
		m.list, _ = m.list.Update(msg)
	case stateImport, stateApply, stateConfirm, stateResult:
		m.viewport, _ = m.viewport.Update(msg)
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateMenu:
		return m.list.View()

	case stateImport:
		return titleStyle.Render("Importing...") + "\n\n" +
			infoStyle.Render("Reading mcp.json files from paths in servers.yaml...")

	case stateApply:
		return titleStyle.Render("Calculating changes...") + "\n\n" +
			infoStyle.Render("Analyzing differences between YAML and current configurations...")

	case stateConfirm:
		title := ""
		prompt := ""
		if m.action == CommandImport {
			title = "Import Preview"
			yesButton := "Yes"
			noButton := "No"
			if m.yesNoIndex == 0 {
				yesButton = successStyle.Render("▶ " + yesButton)
				noButton = infoStyle.Render(noButton)
			} else {
				yesButton = infoStyle.Render(yesButton)
				noButton = errorStyle.Render("▶ " + noButton)
			}
			prompt = "\n" + yesButton + "   " + noButton + "\n" +
				infoStyle.Render("Use ← → to select, Enter to confirm") + "\n" +
				infoStyle.Render("Write this configuration to servers.yaml?")
		} else {
			title = "Apply Preview"
			if strings.Contains(m.viewport.View(), "No changes detected") {
				prompt = "\n" + infoStyle.Render("No changes to apply. Press Enter to return to menu.")
			} else {
				yesButton := "Yes"
				noButton := "No"
				if m.yesNoIndex == 0 {
					yesButton = successStyle.Render("▶ " + yesButton)
					noButton = infoStyle.Render(noButton)
				} else {
					yesButton = infoStyle.Render(yesButton)
					noButton = errorStyle.Render("▶ " + noButton)
				}
				prompt = "\n" + yesButton + "   " + noButton + "\n" +
					infoStyle.Render("Use ← → to select, Enter to confirm") + "\n" +
					infoStyle.Render("Apply these changes?")
			}
		}
		return titleStyle.Render(title) + "\n" +
			m.viewport.View() + "\n" +
			prompt

	case stateResult:
		return titleStyle.Render("Result") + "\n" +
			m.viewport.View() + "\n\n" +
			infoStyle.Render("Press Enter to return to menu")
	default:
		return ""
	}
}

// Commands
type importResult string
type applyPreviewResult string
type actionComplete string
type errMsg struct{ err error }

func (m model) runImport() tea.Cmd {
	return func() tea.Msg {
		content, err := performImport(m.yamlFile)
		if err != nil {
			return errMsg{err}
		}
		return importResult(content)
	}
}

func (m model) runApplyPreview() tea.Cmd {
	return func() tea.Msg {
		preview, err := generateApplyPreview(m.yamlFile)
		if err != nil {
			return errMsg{err}
		}
		return applyPreviewResult(preview)
	}
}

func (m model) executeAction() tea.Cmd {
	return func() tea.Msg {
		if m.action == CommandImport {
			err := os.WriteFile(m.yamlFile, []byte(m.yamlContent), TUIFileMode)
			if err != nil {
				return errMsg{err}
			}
			return actionComplete(successStyle.Render("✓ Successfully wrote configuration to " + m.yamlFile))
		} else {
			result, err := performApply(m.yamlFile)
			if err != nil {
				return errMsg{err}
			}
			return actionComplete(result)
		}
	}
}

// Helper functions
func processSingleClientImport(clientConfig map[string]interface{}, homeDir string) map[string]interface{} {
	pathStr, ok := clientConfig["path"].(string)
	if !ok {
		return map[string]interface{}{
			"path":    "",
			"servers": nil,
		}
	}

	originalPath := pathStr
	validatedPath, err := validateSafePath(pathStr, homeDir)
	if err != nil {
		return map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
	}
	pathStr = validatedPath
	jsonData, err := os.ReadFile(pathStr)
	if err != nil {
		return map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
	}

	var jsonContent map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonContent); err != nil {
		return map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
	}

	mcpServers, ok := jsonContent["mcpServers"].(map[string]interface{})
	if !ok || len(mcpServers) == 0 {
		return map[string]interface{}{
			"path":    originalPath,
			"servers": nil,
		}
	}

	return map[string]interface{}{
		"path":    originalPath,
		"servers": convertMcpServersToYaml(mcpServers),
	}
}

func performImport(yamlFile string) (string, error) {
	yamlContent, err := loadAndValidateYAML(yamlFile)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	clients := yamlContent["clients"].(map[string]interface{})
	outputYaml := map[string]interface{}{
		"clients": make(map[string]interface{}),
	}
	outputClients := outputYaml["clients"].(map[string]interface{})
	for clientName, clientConfig := range clients {
		config, ok := clientConfig.(map[string]interface{})
		if !ok {
			continue
		}
		outputClients[clientName] = processSingleClientImport(config, home)
	}
	yamlBytes, err := yaml.Marshal(outputYaml)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}

func generateApplyPreview(yamlFile string) (string, error) {
	yamlContent, err := loadAndValidateYAML(yamlFile)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	clients := yamlContent["clients"].(map[string]interface{})
	var preview strings.Builder
	hasChanges := false
	for clientName, clientConfig := range clients {
		config, ok := clientConfig.(map[string]interface{})
		if !ok {
			continue
		}
		pathStr, ok := config["path"].(string)
		if !ok {
			continue
		}
		servers := extractClientServers(config)
		if len(servers) == 0 {
			continue
		}
		validatedPath, err := validateSafePath(pathStr, home)
		if err != nil {
			continue
		}
		pathStr = validatedPath
		existing := make(map[string]interface{})
		if data, err := os.ReadFile(pathStr); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &existing)
		}
		existingServers := make(map[string]interface{})
		if existing["mcpServers"] != nil {
			existingServers = existing["mcpServers"].(map[string]interface{})
		}
		clientHasChanges := false
		var clientChanges strings.Builder
		for name := range servers {
			if _, exists := existingServers[name]; !exists {
				clientChanges.WriteString(diffAddStyle.Render(fmt.Sprintf("  + %s", name)) + "\n")
				clientHasChanges = true
			}
		}
		for name := range existingServers {
			if _, exists := servers[name]; !exists {
				clientChanges.WriteString(diffRemoveStyle.Render(fmt.Sprintf("  - %s", name)) + "\n")
				clientHasChanges = true
			}
		}
		for name := range servers {
			if existingServer, exists := existingServers[name]; exists {
				newServerBytes, _ := json.Marshal(servers[name])
				existingServerBytes, _ := json.Marshal(existingServer)
				if string(newServerBytes) != string(existingServerBytes) {
					clientChanges.WriteString(infoStyle.Render(fmt.Sprintf("  ~ %s (updated)", name)) + "\n")
					clientHasChanges = true
				}
			}
		}
		if clientHasChanges {
			preview.WriteString(fmt.Sprintf("\n%s:\n", titleStyle.Render(clientName)))
			preview.WriteString(infoStyle.Render(fmt.Sprintf("  → %s", pathStr)) + "\n")
			preview.WriteString(clientChanges.String())
			hasChanges = true
		}
	}
	if !hasChanges {
		return infoStyle.Render("No changes detected. All configurations are up to date."), nil
	}
	return preview.String(), nil
}

func performApply(yamlFile string) (string, error) {
	yamlContent, err := loadAndValidateYAML(yamlFile)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	clients := yamlContent["clients"].(map[string]interface{})
	var result strings.Builder
	processedCount := 0
	for clientName, clientConfig := range clients {
		config, ok := clientConfig.(map[string]interface{})
		if !ok {
			continue
		}
		pathStr, ok := config["path"].(string)
		if !ok {
			continue
		}
		servers := extractClientServers(config)
		if len(servers) == 0 {
			continue
		}
		validatedPath, err := validateSafePath(pathStr, home)
		if err != nil {
			continue
		}
		pathStr = validatedPath
		existing := make(map[string]interface{})
		if data, err := os.ReadFile(pathStr); err == nil && len(data) > 0 {
			_ = json.Unmarshal(data, &existing)
		}
		existingServers := make(map[string]interface{})
		if existing["mcpServers"] != nil {
			existingServers = existing["mcpServers"].(map[string]interface{})
		}
		hasChanges := false
		for name, server := range servers {
			if existingServer, exists := existingServers[name]; !exists {
				hasChanges = true
				break
			} else {
				newServerBytes, _ := json.Marshal(server)
				existingServerBytes, _ := json.Marshal(existingServer)
				if string(newServerBytes) != string(existingServerBytes) {
					hasChanges = true
					break
				}
			}
		}
		if !hasChanges {
			for name := range existingServers {
				if _, exists := servers[name]; !exists {
					hasChanges = true
					break
				}
			}
		}
		if !hasChanges {
			continue
		}
		if existing["mcpServers"] == nil {
			existing["mcpServers"] = make(map[string]interface{})
		}
		mcpServers := existing["mcpServers"].(map[string]interface{})
		for name, server := range servers {
			mcpServers[name] = server
		}
		if err := os.MkdirAll(filepath.Dir(pathStr), TUIDirectoryMode); err != nil {
			continue
		}
		output, err := json.MarshalIndent(existing, "", "  ")
		if err != nil {
			continue
		}
		if err := os.WriteFile(pathStr, output, TUIFileMode); err != nil {
			continue
		}
		result.WriteString(successStyle.Render(fmt.Sprintf("✓ Updated %s", clientName)) + "\n")
		result.WriteString(infoStyle.Render(fmt.Sprintf("  → %s", pathStr)) + "\n\n")
		processedCount++
	}
	if processedCount > 0 {
		result.WriteString(successStyle.Render(fmt.Sprintf("Successfully processed %d client(s)", processedCount)))
	} else {
		result.WriteString(infoStyle.Render("No changes were applied (all configurations were up to date)"))
	}
	return result.String(), nil
}

func runTUI() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
