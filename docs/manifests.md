# Manifest Management

Manifest management provides declarative application deployment using YAML manifests, similar to the CF CLI.

## Commands

```bash
# Apply a manifest to deploy/update applications
capi manifests apply SPACE_GUID --file manifest.yml [--wait]

# Generate a manifest from an existing application  
capi manifests generate APP_GUID [--output-file manifest.yml]

# Create a diff between current state and proposed manifest
capi manifests diff SPACE_GUID --file manifest.yml
```

## Examples

**Basic manifest (manifest.yml):**
```yaml
applications:
- name: my-app
  memory: 1G
  instances: 2
  buildpacks:
  - go_buildpack
  env:
    NODE_ENV: production
  services:
  - postgres-service
  routes:
  - route: myapp.example.com
```

**Apply manifest:**
```bash
capi manifests apply space-123 --file manifest.yml --wait
```

**Generate manifest from existing app:**
```bash
capi manifests generate app-456 --output-file generated-manifest.yml
```

**Preview changes before applying:**
```bash
capi manifests diff space-123 --file manifest.yml
```

## Output Formats

All manifest commands support multiple output formats:
- `--output table` (default) - Human-readable tables
- `--output json` - JSON for scripting
- `--output yaml` - YAML for configuration management

## CI/CD Pipeline Integration

```bash
#!/bin/bash
# Deploy application using manifest
capi target -o my-org -s production
capi manifests apply $(capi target --space-guid) --file production-manifest.yml --wait

# Enable production features
capi apps features enable my-app revisions
capi spaces features enable production ssh
```

## Development Workflow

```bash
# Development setup
capi target -o my-org -s development

# Deploy app with development manifest
capi manifests apply $(capi target --space-guid) --file dev-manifest.yml

# Enable development features
capi apps features enable my-app ssh
capi spaces features enable development ssh
```

## Troubleshooting

### Common Issues

**Manifest Syntax Errors:**
```bash
Error: Invalid YAML: Manifest contains invalid YAML syntax  
Solution: Validate YAML syntax and check indentation
```

### Getting Help

```bash
# Get help for any command
capi manifests --help

# Get detailed help for subcommands
capi manifests apply --help
```

## API Compatibility

All implemented features are compatible with Cloud Foundry API v3.229.0 and follow the official CF API specification for:

- `/v3/spaces/{guid}/actions/apply_manifest` - Manifest application
- `/v3/apps/{guid}/manifest` - Manifest generation  
- `/v3/spaces/{guid}/manifest_diff` - Manifest diffing

For complete API documentation, see the [Cloud Foundry API v3 specification](https://v3-apidocs.cloudfoundry.org/).