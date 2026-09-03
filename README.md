# Cloud Foundry API v3 Client for Go

A Go client library and CLI for interacting with Cloud Foundry API v3.

[![Go Reference](https://pkg.go.dev/badge/github.com/fivetwenty-io/capi-client.svg)](https://pkg.go.dev/github.com/fivetwenty-io/capi-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/fivetwenty-io/capi-client)](https://goreportcard.com/report/github.com/fivetwenty-io/capi-client)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Features

- **Complete CF API v3 Coverage**: Full support for all Cloud Foundry API v3 resources and operations
- **Type-Safe API**: Generated from official OpenAPI specifications with strong typing
- **Authentication Support**: Multiple auth methods including OAuth2, client credentials, and user credentials
- **Quota Management**: Organization and space quota creation, management, and enforcement
- **Usage Monitoring**: Application and service usage event tracking for billing and analytics
- **Audit Logging**: audit event tracking for security and compliance
- **Application Lifecycle**: Advanced features like revisions, sidecars, and environment management
- **Pagination Handling**: Automatic handling of paginated responses
- **Rate Limiting**: Built-in rate limiting with configurable policies
- **Caching**: Pluggable caching backends (memory, Redis, NATS)
- **CLI Tool**: Full-featured command-line interface for CF operations
- **Testing**: Unit tests, integration tests, and mocks
- **Rich Error Handling**: Detailed error types and context

## Installation

### Go Library

```bash
go get github.com/fivetwenty-io/capi/v3@v3.229.0
```

### CLI Tool

```bash
# Install from source
go install github.com/fivetwenty-io/capi/v3/cmd/capi@v3.229.0

# Or install latest version
go install github.com/fivetwenty-io/capi/v3/cmd/capi@latest

# Or download binary from releases (when available)
curl -L https://github.com/fivetwenty-io/capi/releases/latest/download/capi-linux-amd64 -o capi
chmod +x capi
```

## Quick Start

### Using the Go Client Library

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/fivetwenty-io/capi/v3/pkg/capi"
    "github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

func main() {
    // Create client configuration
    config := &capi.Config{
        APIEndpoint: "https://api.your-cf-domain.com",
        Username:    "username",
        Password:    "password",
    }
    
    // Create client
    client, err := cfclient.New(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // List organizations
    orgs, err := client.Organizations().List(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d organizations:\n", len(orgs.Resources))
    for _, org := range orgs.Resources {
        fmt.Printf("  - %s (%s)\n", org.Name, org.GUID)
    }
}
```

### Using the CLI

```bash
# Login to Cloud Foundry
capi login -a https://api.your-cf-domain.com -u username -p password

# List organizations
capi orgs list

# Target an organization and space
capi target -o my-org -s my-space

# List applications
capi apps list

# Get app details
capi apps get my-app

# Scale an application
capi apps scale my-app --instances 3
```

## Library Documentation

### Authentication

The client supports multiple authentication methods:

#### Username/Password

```go
client, err := cfclient.NewWithPassword("https://api.cf.com", "user", "pass")
```

#### OAuth2 Client Credentials

```go
client, err := cfclient.NewWithClientCredentials("https://api.cf.com", "client-id", "client-secret")
```

#### Access Token

```go
client, err := cfclient.NewWithToken("https://api.cf.com", "access-token")
```

#### Custom Configuration

```go
config := &capi.Config{
    APIEndpoint:   "https://api.cf.com",
    Username:      "user",
    Password:      "pass",
    SkipTLSVerify: false,
    Timeout:       30 * time.Second,
}

client, err := cfclient.New(config)
```

### Resource Operations

#### Organizations

```go
// List organizations
orgs, err := client.Organizations().List(ctx, nil)

// Get specific organization
org, err := client.Organizations().Get(ctx, "org-guid")

// Create organization
createReq := &capi.OrganizationCreate{
    Name: "new-org",
    Metadata: &capi.Metadata{
        Labels: map[string]string{
            "environment": "production",
        },
    },
}
org, err := client.Organizations().Create(ctx, createReq)

// Update organization
updateReq := &capi.OrganizationUpdate{
    Name: capi.String("updated-name"),
}
org, err := client.Organizations().Update(ctx, "org-guid", updateReq)

// Delete organization
job, err := client.Organizations().Delete(ctx, "org-guid")
```

#### Applications

```go
// List applications
apps, err := client.Applications().List(ctx, nil)

// Get application
app, err := client.Applications().Get(ctx, "app-guid")

// Create application
createReq := &capi.ApplicationCreate{
    Name: "my-app",
    Relationships: &capi.ApplicationRelationships{
        Space: &capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
    },
}
app, err := client.Applications().Create(ctx, createReq)

// Start application
app, err := client.Applications().Start(ctx, "app-guid")

// Stop application
app, err := client.Applications().Stop(ctx, "app-guid")

// Scale application
scaleReq := &capi.ProcessScale{
    Instances: capi.Int(5),
    Memory:    capi.String("512M"),
    Disk:      capi.String("1G"),
}
process, err := client.Applications().ScaleProcess(ctx, "app-guid", "web", scaleReq)
```

#### Spaces

```go
// List spaces in organization
params := capi.NewQueryParams().WithFilter("organization_guids", "org-guid")
spaces, err := client.Spaces().List(ctx, params)

// Create space
createReq := &capi.SpaceCreate{
    Name: "dev-space",
    Relationships: &capi.SpaceRelationships{
        Organization: &capi.Relationship{Data: &capi.RelationshipData{GUID: "org-guid"}},
    },
}
space, err := client.Spaces().Create(ctx, createReq)
```

### Pagination

The client automatically handles pagination for list operations:

```go
// Get all pages automatically
allApps := []*capi.Application{}
params := capi.NewQueryParams().WithPerPage(50)

err := client.Applications().ListAll(ctx, params, func(apps *capi.ApplicationList) error {
    allApps = append(allApps, apps.Resources...)
    return nil
})
```

### Error Handling

The client provides rich error information:

```go
app, err := client.Applications().Get(ctx, "invalid-guid")
if err != nil {
    if capiErr, ok := err.(*capi.Error); ok {
        fmt.Printf("CF Error %d: %s\n", capiErr.Status, capiErr.Title)
        for _, detail := range capiErr.Errors {
            fmt.Printf("  - %s: %s\n", detail.Code, detail.Detail)
        }
    }
}
```

### Caching

Enable caching for improved performance:

```go
config := &capi.Config{
    APIEndpoint: "https://api.cf.com",
    Username:    "user",
    Password:    "pass",
    Cache: &capi.CacheConfig{
        Type: "memory",
        TTL:  5 * time.Minute,
    },
}

client, err := cfclient.New(config)
```

## CLI Documentation

### Installation and Login

```bash
# Login with prompts
capi login

# Login with flags
capi login -a https://api.cf.com -u user -p password

# Login with SSO
capi login -a https://api.cf.com --sso

# Skip SSL validation (not recommended for production)
capi login -a https://api.cf.com --skip-ssl-validation
```

### Targeting

```bash
# Show current target
capi target

# Target organization
capi target -o my-org

# Target organization and space
capi target -o my-org -s my-space
```

### Organizations

```bash
# List organizations
capi orgs list

# Get organization details
capi orgs get my-org

# Create organization
capi orgs create new-org

# Update organization
capi orgs update my-org --name updated-name

# Delete organization
capi orgs delete my-org
```

### Spaces

```bash
# List spaces
capi spaces list

# List spaces in specific organization
capi spaces list -o my-org

# Create space
capi spaces create dev-space -o my-org

# Delete space
capi spaces delete dev-space
```

### Applications

```bash
# List applications
capi apps list

# Get application details
capi apps get my-app

# Create application
capi apps create my-app

# Start application
capi apps start my-app

# Stop application
capi apps stop my-app

# Scale application
capi apps scale my-app --instances 3 --memory 512M

# Delete application
capi apps delete my-app
```

### Quota Management

```bash
# Organization quotas
capi org-quotas list
capi org-quotas get production-quota
capi org-quotas create --name dev-quota --total-memory 2048 --instances 10
capi org-quotas update production-quota --total-memory 4096
capi org-quotas apply production-quota my-org-1 my-org-2
capi org-quotas delete old-quota

# Space quotas
capi space-quotas list
capi space-quotas list --org my-org
capi space-quotas create --name dev-space-quota --org my-org --total-memory 1024
capi space-quotas apply dev-space-quota my-space-1 my-space-2
capi space-quotas remove dev-space-quota my-space-1
```

### Usage Monitoring

```bash
# Application usage events
capi app-usage-events list
capi app-usage-events list --app-name my-app --start-time 2023-01-01T00:00:00Z
capi app-usage-events get event-guid
capi app-usage-events purge-and-reseed

# Service usage events
capi service-usage-events list
capi service-usage-events get event-guid
capi service-usage-events purge-and-reseed

# Audit events
capi audit-events list
capi audit-events list --target-ids app-guid
capi audit-events get event-guid
```

### Application Lifecycle

```bash
# Revisions
capi revisions get revision-guid
capi revisions get-env revision-guid
capi revisions update revision-guid --metadata team=backend,version=1.2.0

# Sidecars
capi sidecars get sidecar-guid
capi sidecars list-for-process process-guid
capi sidecars update sidecar-guid --name new-name --command "./new-command"
capi sidecars delete sidecar-guid

# Environment variable groups
capi env-var-groups get running
capi env-var-groups get staging
capi env-var-groups update running LOG_LEVEL=debug TIMEOUT=60
capi env-var-groups update staging BUILD_CACHE=true

# Resource matches
capi resource-matches create resource-list.json
```

### UAA User Management

The CLI includes UAA (User Account and Authentication) user management functionality:

```bash
# Set UAA endpoint
capi uaa target https://uaa.your-cf-domain.com

# Authenticate with client credentials
capi uaa get-client-credentials-token --client-id admin --client-secret admin-secret

# Authenticate with username/password
capi uaa get-password-token --username admin --password admin-pass --client-id cf

# Create a user
capi uaa create-user john.doe --email john.doe@example.com --password SecurePass123!

# List users with filtering
capi uaa list-users --filter 'email co "example.com"'

# Get user details
capi uaa get-user john.doe

# Create a group
capi uaa create-group developers --description "Development team members"

# Add user to group
capi uaa add-member developers john.doe

# Create OAuth client
capi uaa create-client my-app --secret app-secret --authorized-grant-types client_credentials

# Get current user info
capi uaa userinfo

# Direct UAA API access
capi uaa curl /users --method GET
```

For UAA documentation, see [docs/uaa-commands.md](./docs/uaa-commands.md).

### Output Formats

The CLI supports multiple output formats:

```bash
# Table format (default)
capi orgs list

# JSON format
capi orgs list --output json

# YAML format
capi orgs list --output yaml
```

### Configuration

```bash
# Show configuration
capi config show

# Set configuration value
capi config set output json

# Unset configuration value
capi config unset output

# Clear all configuration
capi config clear
```

## Examples

See the [examples](./examples) directory for examples:

- [Basic Usage](./examples/basic/)
- [Authentication](./examples/auth/)
- [Application Management](./examples/apps/)
- [Service Management](./examples/services/)
- [Quota Management](./examples/quota-management/)
- [Usage Monitoring](./examples/usage-monitoring/)
- [Lifecycle Management](./examples/lifecycle-management/)
- [Advanced Usage](./examples/advanced/)

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/fivetwenty-io/capi-client.git
cd capi-client

# Install dependencies
go mod download

# Run tests
make test

# Run linting
make lint

# Build CLI
make build
```

### Testing

```bash
# Run unit tests
make test

# Run integration tests (requires CF environment)
make test-integration

# Run tests with coverage
make test-coverage
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/fivetwenty-io/capi-client/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fivetwenty-io/capi-client/discussions)
- **Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/fivetwenty-io/capi-client)

## Changelog

See [CHANGELOG.md](./CHANGELOG.md) for release notes and version history.
