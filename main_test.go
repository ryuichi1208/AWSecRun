package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// モック実装
// ============

// MockLogger はLogger interfaceのモック実装
type MockLogger struct {
	Logs []struct {
		Level   string
		Message string
		Data    interface{}
	}
}

// Log はログを記録するだけでOutputには書き込まない
func (l *MockLogger) Log(level, message string, data interface{}) {
	l.Logs = append(l.Logs, struct {
		Level   string
		Message string
		Data    interface{}
	}{level, message, data})
}

// MockSecretManager はSecretManager interfaceのモック実装
type MockSecretManager struct {
	Secrets map[string]string
	Calls   []string
	Error   error
}

// GetSecret はモックされたシークレットを返す
func (m *MockSecretManager) GetSecret(secretName string) (string, error) {
	m.Calls = append(m.Calls, secretName)
	if m.Error != nil {
		return "", m.Error
	}
	if secret, ok := m.Secrets[secretName]; ok {
		return secret, nil
	}
	return "", fmt.Errorf("secret not found: %s", secretName)
}

// MockCommandRunner はCommandRunner interfaceのモック実装
type MockCommandRunner struct {
	ExecutedCommands []struct {
		Path string
		Args []string
		Env  []string
	}
	ReturnError error
}

// Run はコマンド実行をモックする
func (r *MockCommandRunner) Run(commandPath string, args []string, env []string) error {
	r.ExecutedCommands = append(r.ExecutedCommands, struct {
		Path string
		Args []string
		Env  []string
	}{commandPath, args, env})
	return r.ReturnError
}

// 既存のテスト
// ===========

func TestParseSecretJSON(t *testing.T) {
	tests := []struct {
		name         string
		secretString string
		want         map[string]string
		wantErr      bool
	}{
		{
			name:         "Valid JSON",
			secretString: `{"key1":"value1","key2":"value2"}`,
			want:         map[string]string{"key1": "value1", "key2": "value2"},
			wantErr:      false,
		},
		{
			name:         "Non-JSON string",
			secretString: "just a string",
			want:         map[string]string{"secret": "just a string"},
			wantErr:      false,
		},
		{
			name:         "Empty string",
			secretString: "",
			want:         map[string]string{"secret": ""},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSecretJSON(tt.secretString)

			// エラー確認
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSecretJSON() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSecretJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 結果確認
			if len(got) != len(tt.want) {
				t.Errorf("parseSecretJSON() got = %v, want %v", got, tt.want)
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseSecretJSON() got[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestLogJSON(t *testing.T) {
	// 標準出力をキャプチャする
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// テスト終了後に標準出力を元に戻す
	defer func() {
		os.Stdout = oldStdout
	}()

	// テストデータ
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// logJSON関数を呼び出す
	logJSON("info", "Test message", testData)

	// 標準出力の内容を取得
	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// JSONとしてパースできることを確認
	var logEntry LogEntry
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
		return
	}

	// 値を確認
	if logEntry.Level != "info" {
		t.Errorf("Level = %v, want %v", logEntry.Level, "info")
	}
	if logEntry.Message != "Test message" {
		t.Errorf("Message = %v, want %v", logEntry.Message, "Test message")
	}

	// データ部分を検証
	dataMap, ok := logEntry.Data.(map[string]interface{})
	if !ok {
		t.Errorf("Data is not a map: %v", logEntry.Data)
		return
	}

	// 値を個別チェック
	if dataMap["key1"] != "value1" {
		t.Errorf("Data[key1] = %v, want %v", dataMap["key1"], "value1")
	}
	if dataMap["key2"] != "value2" {
		t.Errorf("Data[key2] = %v, want %v", dataMap["key2"], "value2")
	}
}

func TestUsage(t *testing.T) {
	// 元のosArgsを保存
	originalArgs := os.Args

	// テスト終了後に復元
	defer func() {
		os.Args = originalArgs
	}()

	// 引数なしでrunを呼び出す
	os.Args = []string{"main"}
	err := run()

	// エラーが返されることを確認
	if err == nil {
		t.Error("Expected error for missing arguments, got nil")
		return
	}

	// エラーメッセージに"Usage:"が含まれることを確認
	if !strings.Contains(err.Error(), "Usage:") {
		t.Errorf("Expected usage error message, got: %v", err)
	}
}

// モックを使った追加のテスト
// ===================

func TestApplication_Run_WithMocks(t *testing.T) {
	// モックの準備
	mockLogger := &MockLogger{}
	mockSecretManager := &MockSecretManager{
		Secrets: map[string]string{
			"db-creds": `{"DB_USER":"admin","DB_PASSWORD":"secure123"}`,
		},
	}
	mockRunner := &MockCommandRunner{}

	// テスト用アプリケーション
	app := &Application{
		Logger:        mockLogger,
		SecretManager: mockSecretManager,
		CommandRunner: mockRunner,
		Args:          []string{"program", "/usr/bin/env", "--key", "db-creds"},
	}

	// 実行
	err := app.Run()

	// 検証
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// シークレットマネージャーの呼び出し確認
	if len(mockSecretManager.Calls) != 1 || mockSecretManager.Calls[0] != "db-creds" {
		t.Errorf("Expected call to GetSecret with 'db-creds', got: %v", mockSecretManager.Calls)
	}

	// コマンド実行確認
	if len(mockRunner.ExecutedCommands) != 1 {
		t.Fatalf("Expected 1 command execution, got: %d", len(mockRunner.ExecutedCommands))
	}

	cmd := mockRunner.ExecutedCommands[0]
	if cmd.Path != "/usr/bin/env" {
		t.Errorf("Expected command '/usr/bin/env', got: '%s'", cmd.Path)
	}

	// 環境変数の確認
	foundDBUser := false
	foundDBPassword := false
	for _, env := range cmd.Env {
		if env == "DB_USER=admin" {
			foundDBUser = true
		}
		if env == "DB_PASSWORD=secure123" {
			foundDBPassword = true
		}
	}

	if !foundDBUser {
		t.Error("Expected DB_USER environment variable")
	}
	if !foundDBPassword {
		t.Error("Expected DB_PASSWORD environment variable")
	}

	// ログの確認
	if len(mockLogger.Logs) < 3 {
		t.Fatalf("Expected at least 3 log entries, got: %d", len(mockLogger.Logs))
	}

	// シークレット取得ログ
	fetchLogFound := false
	for _, log := range mockLogger.Logs {
		if log.Level == "info" && strings.Contains(log.Message, "Fetching secret") {
			fetchLogFound = true
			break
		}
	}
	if !fetchLogFound {
		t.Error("Expected log about fetching secret")
	}

	// 成功ログ
	successLogFound := false
	for _, log := range mockLogger.Logs {
		if log.Level == "info" && strings.Contains(log.Message, "Command executed successfully") {
			successLogFound = true
			break
		}
	}
	if !successLogFound {
		t.Error("Expected log about successful execution")
	}
}

func TestApplication_Run_SecretManagerError(t *testing.T) {
	// モックの準備
	mockLogger := &MockLogger{}
	mockSecretManager := &MockSecretManager{
		Error: fmt.Errorf("connection error"),
	}
	mockRunner := &MockCommandRunner{}

	// テスト用アプリケーション
	app := &Application{
		Logger:        mockLogger,
		SecretManager: mockSecretManager,
		CommandRunner: mockRunner,
		Args:          []string{"program", "/bin/ls", "--key", "some-secret"},
	}

	// 実行
	err := app.Run()

	// エラーが発生することを確認
	if err == nil {
		t.Fatal("Expected error from SecretManager, got nil")
	}

	// エラーメッセージの確認
	if !strings.Contains(err.Error(), "failed to get secret") {
		t.Errorf("Expected error about failing to get secret, got: %v", err)
	}

	// シークレットマネージャーの呼び出し確認
	if len(mockSecretManager.Calls) != 1 || mockSecretManager.Calls[0] != "some-secret" {
		t.Errorf("Expected call to GetSecret with 'some-secret', got: %v", mockSecretManager.Calls)
	}

	// コマンドが実行されていないことを確認
	if len(mockRunner.ExecutedCommands) > 0 {
		t.Errorf("Expected no command execution, got: %d", len(mockRunner.ExecutedCommands))
	}
}

func TestApplication_Run_MultipleSecrets(t *testing.T) {
	// モックの準備
	mockLogger := &MockLogger{}
	mockSecretManager := &MockSecretManager{
		Secrets: map[string]string{
			"api-keys":  `{"API_KEY":"xyz123","API_SECRET":"abc456"}`,
			"db-config": `{"DB_HOST":"localhost","DB_PORT":"5432"}`,
		},
	}
	mockRunner := &MockCommandRunner{}

	// テスト用アプリケーション
	app := &Application{
		Logger:        mockLogger,
		SecretManager: mockSecretManager,
		CommandRunner: mockRunner,
		Args:          []string{"program", "/bin/echo", "test", "--key", "api-keys", "--key", "db-config"},
	}

	// 実行
	err := app.Run()

	// 検証
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// シークレットマネージャーの呼び出し確認
	if len(mockSecretManager.Calls) != 2 {
		t.Fatalf("Expected 2 calls to GetSecret, got: %d", len(mockSecretManager.Calls))
	}

	// 呼び出し順序の確認
	if mockSecretManager.Calls[0] != "api-keys" || mockSecretManager.Calls[1] != "db-config" {
		t.Errorf("Expected calls to 'api-keys' then 'db-config', got: %v", mockSecretManager.Calls)
	}

	// コマンドの確認
	if len(mockRunner.ExecutedCommands) != 1 {
		t.Fatalf("Expected 1 command execution, got: %d", len(mockRunner.ExecutedCommands))
	}

	cmd := mockRunner.ExecutedCommands[0]
	if cmd.Path != "/bin/echo" || len(cmd.Args) != 1 || cmd.Args[0] != "test" {
		t.Errorf("Expected command '/bin/echo test', got: '%s %v'", cmd.Path, cmd.Args)
	}

	// すべての環境変数が設定されたか確認
	envVars := []string{"API_KEY", "API_SECRET", "DB_HOST", "DB_PORT"}
	for _, envVar := range envVars {
		found := false
		for _, env := range cmd.Env {
			if strings.HasPrefix(env, envVar+"=") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected environment variable %s to be set", envVar)
		}
	}
}

func TestApplication_Run_CommandError(t *testing.T) {
	// モックの準備
	mockLogger := &MockLogger{}
	mockSecretManager := &MockSecretManager{}
	mockRunner := &MockCommandRunner{
		ReturnError: fmt.Errorf("command execution failed"),
	}

	// テスト用アプリケーション
	app := &Application{
		Logger:        mockLogger,
		SecretManager: mockSecretManager,
		CommandRunner: mockRunner,
		Args:          []string{"program", "/bin/false"},
	}

	// 実行
	err := app.Run()

	// エラーが発生することを確認
	if err == nil {
		t.Fatal("Expected error from CommandRunner, got nil")
	}

	// エラーメッセージの確認
	if !strings.Contains(err.Error(), "Command execution error") {
		t.Errorf("Expected error about command execution, got: %v", err)
	}

	// エラーログの確認
	errorLogFound := false
	for _, log := range mockLogger.Logs {
		if log.Level == "error" && strings.Contains(log.Message, "Command execution failed") {
			errorLogFound = true
			break
		}
	}
	if !errorLogFound {
		t.Error("Expected error log about command execution")
	}
}
