# How to Import and Use the CAPI Client

## Installation

### Install as a Library

```bash
go get github.com/fivetwenty-io/capi/v3@v3.229.1
```

### Install the CLI Tool

```bash
go install github.com/fivetwenty-io/capi/v3/cmd/capi@v3.229.1
```

After installation, the `capi` command will be available in your `$GOPATH/bin` or `$HOME/go/bin`.

## Basic Usage

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
    // Configure the client
    config := &capi.Config{
        APIEndpoint: "https://api.cf.example.com",
        
        // Authentication options (choose one):
        // Option 1: Access token
        AccessToken: "your-access-token",
        
        // Option 2: Username/Password
        // Username: "user@example.com",
        // Password: "password",
        
        // Option 3: Client credentials
        // ClientID:     "your-client-id",
        // ClientSecret: "your-client-secret",
    }
    
    // Create the client
    client, err := cfclient.New(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the client
    ctx := context.Background()
    
    // List organizations
    orgsResp, err := client.Organizations().List(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d organizations\n", len(orgsResp.Resources))
    
    // List spaces
    spacesResp, err := client.Spaces().List(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d spaces\n", len(spacesResp.Resources))
    
    // Get an app by GUID
    app, err := client.Apps().Get(ctx, "app-guid")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("App: %s\n", app.Name)
}
```

## Advanced Features

### Pagination

```go
// Use query parameters for pagination
params := capi.NewQueryParams()
params.PerPage = 50
params.OrderBy = "name"

orgsResp, err := client.Organizations().List(ctx, params)
```

### Filtering

```go
params := capi.NewQueryParams()
params.AddFilter("name", "my-org")
spacesResp, err := client.Spaces().List(ctx, params)
```

## Versioning

This module uses semantic versioning aligned with the Cloud Foundry API v3 specification version it implements.

Current version: **v3.229.1** (implements CF API v3.229.0)

To import a specific version:
```bash
go get github.com/fivetwenty-io/capi/v3@v3.229.1
```

For the latest version:
```bash
go get github.com/fivetwenty-io/capi/v3@latest
```

## Package Structure

- `pkg/capi` - Core types and interfaces
- `pkg/cfclient` - Client factory and implementation