package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTest() func() {
	originalArgs := os.Args
	return func() {
		os.Args = originalArgs
	}
}

type testMocks struct {
	originalRunTUI       func()
	originalApplyConfig  func(string)
	originalImportConfig func(string)
	originalOsExit       func(int)
}

func (m *testMocks) setup() {
	m.originalRunTUI = runTUIFunc
	m.originalApplyConfig = applyConfigFunc
	m.originalImportConfig = importConfigFunc
	m.originalOsExit = osExit
}

func (m *testMocks) restore() {
	runTUIFunc = m.originalRunTUI
	applyConfigFunc = m.originalApplyConfig
	importConfigFunc = m.originalImportConfig
	osExit = m.originalOsExit
}

// TestMain_NoArguments_LaunchesTUI TUIモードのデフォルト起動テスト
func TestMain_NoArguments_LaunchesTUI(t *testing.T) {
	defer setupTest()()
	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	// 引数をTUIモード用に設定
	os.Args = []string{"mcp-setup"}
	tuiLaunched := false
	runTUIFunc = func() { tuiLaunched = true }

	main()

	// TUIが起動されたことを確認
	assert.True(t, tuiLaunched, "引数なしの場合、TUIが起動されるべき")
}

// TestMain_ApplyCommand_CallsApplyConfig applyコマンドテスト
func TestMain_ApplyCommand_CallsApplyConfig(t *testing.T) {
	defer setupTest()()

	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	// applyコマンド用引数設定
	os.Args = []string{"mcp-setup", "apply", "test.yaml"}

	applyCalled := false
	var applyFile string
	applyConfigFunc = func(file string) {
		applyCalled = true
		applyFile = file
	}

	main()

	assert.True(t, applyCalled, "applyConfigが呼ばれるべき")
	assert.Equal(t, "test.yaml", applyFile, "正しいファイル名が渡されるべき")
}

// TestMain_ImportCommand_CallsImportConfig importコマンドテスト
func TestMain_ImportCommand_CallsImportConfig(t *testing.T) {
	defer setupTest()()

	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	os.Args = []string{"mcp-setup", "import", "test.yaml"}

	importCalled := false
	var importFile string
	importConfigFunc = func(file string) {
		importCalled = true
		importFile = file
	}

	main()

	assert.True(t, importCalled, "importConfigが呼ばれるべき")
	assert.Equal(t, "test.yaml", importFile, "正しいファイル名が渡されるべき")
}

// TestMain_UnknownCommand_ExitsWithError 不明なコマンドエラーテスト
func TestMain_UnknownCommand_ExitsWithError(t *testing.T) {
	defer setupTest()()

	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	// 不明なコマンド設定
	os.Args = []string{"mcp-setup", "unknown"}

	exitCalled := false
	exitCode := 0
	osExit = func(code int) {
		exitCalled = true
		exitCode = code
		panic("os.Exit called")
	}

	assert.Panics(t, func() { main() }, "不明なコマンドでos.Exitが呼ばれるべき")
	assert.True(t, exitCalled, "os.Exitが呼ばれるべき")
	assert.Equal(t, 1, exitCode, "終了コードは1であるべき")
}

// TestMain_ApplyWithoutFile_ExitsWithError ファイル引数なしエラーテスト
func TestMain_ApplyWithoutFile_ExitsWithError(t *testing.T) {
	defer setupTest()()

	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	// ファイル引数なしのapplyコマンド
	os.Args = []string{"mcp-setup", "apply"}

	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
		panic("os.Exit called")
	}

	assert.Panics(t, func() { main() })
	assert.True(t, exitCalled, "ファイル引数なしでos.Exitが呼ばれるべき")
}

// TestMain_ImportWithoutFile_ExitsWithError importファイル引数なしエラーテスト
func TestMain_ImportWithoutFile_ExitsWithError(t *testing.T) {
	defer setupTest()()

	mocks := &testMocks{}
	mocks.setup()
	defer mocks.restore()

	// ファイル引数なしのimportコマンド
	os.Args = []string{"mcp-setup", "import"}

	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
		panic("os.Exit called")
	}

	assert.Panics(t, func() { main() })
	assert.True(t, exitCalled, "ファイル引数なしでos.Exitが呼ばれるべき")
}

// TestCommandRunner_Interface CommandRunnerインターフェースのテスト
func TestCommandRunner_Interface(t *testing.T) {
	defer setupTest()()

	// テスト用のCommandRunner実装
	testRunner := &CLICommandRunner{}

	// インターフェースが実装されていることを確認
	var _ CommandRunner = testRunner

	// runCommandメソッドが存在することを確認
	assert.NotNil(t, testRunner.runCommand)
}

// TestCLICommandRunner_RunCommand CLICommandRunnerのテスト
func TestCLICommandRunner_RunCommand(t *testing.T) {
	defer setupTest()()

	runner := &CLICommandRunner{}

	// モックの設定
	var capturedCommand string
	var capturedArg string
	originalApplyConfig := applyConfigFunc
	originalImportConfig := importConfigFunc

	applyConfigFunc = func(arg string) {
		capturedCommand = "apply"
		capturedArg = arg
	}
	importConfigFunc = func(arg string) {
		capturedCommand = "import"
		capturedArg = arg
	}

	defer func() {
		applyConfigFunc = originalApplyConfig
		importConfigFunc = originalImportConfig
	}()

	// applyコマンドのテスト
	runner.runCommand("apply", "test.yaml")
	assert.Equal(t, "apply", capturedCommand)
	assert.Equal(t, "test.yaml", capturedArg)

	// importコマンドのテスト
	runner.runCommand("import", "config.yaml")
	assert.Equal(t, "import", capturedCommand)
	assert.Equal(t, "config.yaml", capturedArg)
}

// TestBaseProcessor_GetHomeDir BaseProcessorのホームディレクトリ取得テスト
func TestBaseProcessor_GetHomeDir(t *testing.T) {
	processor := &BaseProcessor{}

	homeDir, err := processor.getHomeDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, homeDir)
}

// TestBaseProcessor_LoadAndValidateYAML BaseProcessorのYAML読み込みテスト
func TestBaseProcessor_LoadAndValidateYAML(t *testing.T) {
	processor := &BaseProcessor{}

	// テスト用の一時ファイルを作成
	tmpFile, err := os.CreateTemp("", "test_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// テスト用のYAMLデータを書き込み
	yamlContent := `clients:
  test:
    path: "/test/path"
    servers:
      test-server:
        command: "test"`
	_, err = tmpFile.WriteString(yamlContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// loadAndValidateYAMLのテスト
	result, err := processor.loadAndValidateYAML(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "clients")
}
