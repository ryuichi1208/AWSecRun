# AWSecRun

AWSecRun is a tool that retrieves secrets from AWS Secrets Manager, sets them as environment variables, and executes commands.

## Features

- Execute specified commands
- Retrieve secrets from AWS Secrets Manager
- Parse secrets as JSON and set them as environment variables
- Support for multiple secrets retrieval
- Structured JSON logging

## Installation

### Prerequisites

- Go 1.16 or higher
- AWS access permissions (for Secrets Manager)

### Build Instructions

```bash
# Clone the repository
git clone https://github.com/ryuichi1208/AWSecRun.git
cd AWSecRun

# Install dependencies
go mod tidy

# Build
go build -o awsecrun
```

## Usage

```bash
# Basic usage
./awsecrun <command_path> [args...] [--key SECRET_NAME...]

# Example: Set secrets as environment variables and display all environment variables
./awsecrun /usr/bin/env --key database-credentials

# Example: Retrieve multiple secrets
./awsecrun /bin/bash -c "echo $DB_USER" --key db-creds --key api-keys
```

## Secret Format

Secrets should be stored in JSON format, for example:

```json
{
  "DB_USER": "admin",
  "DB_PASSWORD": "secure123",
  "DB_HOST": "localhost",
  "DB_PORT": "5432"
}
```

If the secret is not in JSON format, it will be set as an environment variable named `secret`.

## AWS Configuration

AWS credentials must be configured using one of the following methods:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM roles (EC2, ECS, Lambda, etc.)

## Logging

All logs are output in JSON format with the following fields:

- `timestamp`: ISO 8601 formatted timestamp
- `level`: Log level (info, error)
- `message`: Log message
- `data`: Additional data (object)

## Error Handling

- Secret retrieval errors: The program exits immediately and displays error messages
- Command execution errors: Displays error code and detailed message

## Architecture

AWSecRun uses an interface-based design:

- `Logger` - Responsible for logging
- `SecretManager` - Responsible for retrieving secrets
- `CommandRunner` - Responsible for executing commands

This design improves testability and extensibility.

## Testing

```bash
# Run tests
go test

# Generate coverage report
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

Released under the MIT License.
