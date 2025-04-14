package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

// Logger defines the interface for logging
type Logger interface {
	Log(level, message string, data interface{})
}

// JSONLogger implements Logger with JSON format output
type JSONLogger struct {
	Output *os.File
}

// Log outputs a structured log entry in JSON format
func (l *JSONLogger) Log(level, message string, data interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		fmt.Fprintf(os.Stderr, "Error marshaling log: %v\n", err)
		return
	}

	fmt.Fprintln(l.Output, string(jsonBytes))
}

// NewJSONLogger creates a new JSON logger with stdout as default output
func NewJSONLogger() *JSONLogger {
	return &JSONLogger{
		Output: os.Stdout,
	}
}

// SecretManager defines the interface for retrieving secrets
type SecretManager interface {
	GetSecret(secretName string) (string, error)
}

// AWSSecretManager implements SecretManager using AWS SecretsManager
type AWSSecretManager struct {
	ctx context.Context
}

// NewAWSSecretManager creates a new AWSSecretManager
func NewAWSSecretManager() *AWSSecretManager {
	return &AWSSecretManager{
		ctx: context.Background(),
	}
}

// GetSecret retrieves a secret from AWS Secrets Manager
func (sm *AWSSecretManager) GetSecret(secretName string) (string, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(sm.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create a Secrets Manager client
	svc := secretsmanager.NewFromConfig(cfg)

	// Get the secret value
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(sm.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret value: %w", err)
	}

	// Get the secret string
	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	}

	return secretString, nil
}

// CommandRunner defines the interface for running commands
type CommandRunner interface {
	Run(commandPath string, args []string, env []string) error
}

// DefaultCommandRunner implements CommandRunner using os/exec
type DefaultCommandRunner struct {
	Stdout *os.File
	Stderr *os.File
	Stdin  *os.File
}

// NewCommandRunner creates a new DefaultCommandRunner
func NewCommandRunner() *DefaultCommandRunner {
	return &DefaultCommandRunner{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
}

// Run executes a command with the given args and environment
func (cr *DefaultCommandRunner) Run(commandPath string, args []string, env []string) error {
	cmd := exec.Command(commandPath, args...)
	cmd.Stdout = cr.Stdout
	cmd.Stderr = cr.Stderr
	cmd.Stdin = cr.Stdin
	cmd.Env = env

	return cmd.Run()
}

// Application contains all dependencies
type Application struct {
	Logger        Logger
	SecretManager SecretManager
	CommandRunner CommandRunner
	Args          []string
}

// NewApplication creates a new Application with default implementations
func NewApplication(args []string) *Application {
	return &Application{
		Logger:        NewJSONLogger(),
		SecretManager: NewAWSSecretManager(),
		CommandRunner: NewCommandRunner(),
		Args:          args,
	}
}

// parseSecretJSON parses a JSON secret string and returns a map of key-value pairs
func parseSecretJSON(secretString string) (map[string]string, error) {
	secretMap := make(map[string]string)

	// Try to parse as JSON first
	err := json.Unmarshal([]byte(secretString), &secretMap)
	if err != nil {
		// If not JSON, just use the raw string as the value
		return map[string]string{"secret": secretString}, nil
	}

	return secretMap, nil
}

// Run executes the command with arguments and environment variables
func (app *Application) Run() error {
	if len(app.Args) < 2 {
		return fmt.Errorf("Usage: go run main.go <command_path> [args...] [--key SECRET_NAME]")
	}

	commandPath := app.Args[1]
	args := []string{}
	envVars := map[string]string{}

	// Parse arguments to separate normal args from --key options
	for i := 2; i < len(app.Args); i++ {
		if app.Args[i] == "--key" && i+1 < len(app.Args) {
			secretName := app.Args[i+1]
			app.Logger.Log("info", "Fetching secret from AWS Secrets Manager", map[string]string{"secretName": secretName})

			secretString, err := app.SecretManager.GetSecret(secretName)
			if err != nil {
				return fmt.Errorf("failed to get secret %s: %w", secretName, err)
			}

			secretMap, err := parseSecretJSON(secretString)
			if err != nil {
				return fmt.Errorf("failed to parse secret as JSON: %w", err)
			}

			// Add all key-value pairs from the secret to environment variables
			secretKeys := make([]string, 0, len(secretMap))
			for k, v := range secretMap {
				envVars[k] = v
				secretKeys = append(secretKeys, k)
			}
			app.Logger.Log("info", "Retrieved secret keys", map[string]interface{}{"keys": secretKeys})

			i++ // Skip the next argument as it's the secret name
		} else {
			args = append(args, app.Args[i])
		}
	}

	// Set environment variables from the parent process
	env := os.Environ()

	// Add or override environment variables from AWS Secrets Manager
	for k, v := range envVars {
		env = append(env, k+"="+v)
	}

	app.Logger.Log("info", "Executing command", map[string]interface{}{
		"commandPath": commandPath,
		"args":        args,
	})

	err := app.CommandRunner.Run(commandPath, args, env)
	if err != nil {
		app.Logger.Log("error", "Command execution failed", map[string]string{"error": err.Error()})
		return fmt.Errorf("Command execution error: %w", err)
	}

	app.Logger.Log("info", "Command executed successfully", nil)
	return nil
}

// logJSON is a helper function for backward compatibility
func logJSON(level, message string, data interface{}) {
	logger := NewJSONLogger()
	logger.Log(level, message, data)
}

// run is a helper function for backward compatibility
func run() error {
	app := NewApplication(os.Args)
	return app.Run()
}

func main() {
	if err := run(); err != nil {
		logJSON("error", err.Error(), nil)
		os.Exit(1)
	}
}
