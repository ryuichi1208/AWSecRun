# AWSecRun

AWSecRun is a tool that retrieves secrets from AWS Secrets Manager, sets them as environment variables, and executes commands.

## Quick Start

```bash
# Install
go install github.com/ryuichi1208/AWSecRun@latest

# Or use Docker
docker run ryuichi1208/awsecrun:latest

# Usage
awsecrun <command_path> [args...] [--key SECRET_NAME...]

# Example
awsecrun /usr/bin/env --key database-credentials
```

## Features

- Retrieve secrets from AWS Secrets Manager
- Set secrets as environment variables (parses JSON)
- Support for multiple secrets
- JSON logging
- Interface-based design for easy testing

## AWS Configuration

AWS credentials can be configured via environment variables, shared credentials file, or IAM roles.

## Docker

```bash
docker run -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY ryuichi1208/awsecrun /bin/bash -c "echo $DB_PASSWORD" --key my-secret
```

## Documentation

For more information on usage, architecture, and contributing, see the [full documentation](https://github.com/ryuichi1208/AWSecRun/wiki).

## License

MIT
