# Space Features Management

Space features control functionality at the space level, affecting all applications within the space.

## Commands

```bash
# List all features for a space
capi spaces features list SPACE_NAME_OR_GUID

# Get details for a specific space feature
capi spaces features get SPACE_NAME_OR_GUID FEATURE_NAME

# Enable a feature for a space
capi spaces features enable SPACE_NAME_OR_GUID FEATURE_NAME

# Disable a feature for a space
capi spaces features disable SPACE_NAME_OR_GUID FEATURE_NAME
```

## Examples

```bash
# List features for a space
capi spaces features list development

# Enable SSH for all apps in a space
capi spaces features enable development ssh

# Disable SSH for a space
capi spaces features disable development ssh
```

## Common Space Features

- `ssh` - SSH access for all applications in the space

## Output Formats

All commands support three output formats:

```bash
# Human-readable table (default)
capi spaces features list development

# JSON for automation and scripting
capi spaces features list development --output json

# YAML for configuration management
capi spaces features list development --output yaml
```

## Resource Resolution

Commands support both GUIDs and names for user convenience:

```bash
# Using GUID
capi spaces features enable space-123-456 ssh

# Using name (searches in targeted org)
capi spaces features enable development ssh
```

## Error Handling

All commands provide clear error messages and helpful suggestions:

```bash
$ capi spaces features enable non-existent-space ssh
Error: space 'non-existent-space' not found
```

## Integration Examples

### CI/CD Pipeline Integration

```bash
#!/bin/bash
# Deploy application using manifest
capi target -o my-org -s production
capi manifests apply $(capi target --space-guid) --file production-manifest.yml --wait

# Enable production features
capi spaces features enable production ssh
```

### Development Workflow

```bash
# Development setup
capi target -o my-org -s development

# Enable development features
capi spaces features enable development ssh
```

## Security Considerations

- **Permissions**: All operations respect Cloud Foundry RBAC permissions
- **Authentication**: All operations require proper CF authentication
- **Audit Trail**: Operations are logged in CF audit events

## Troubleshooting

### Common Issues

**Permission Denied:**
```bash
Error: insufficient permissions to perform operation
Solution: Ensure you have appropriate space/org roles
```

**Resource Not Found:**
```bash  
Error: space 'my-space' not found
Solution: Check space name and ensure you're targeting the correct org
```

### Getting Help

```bash
# Get help for any command
capi spaces features --help  

# Get detailed help for subcommands
capi spaces features enable --help
```

## API Compatibility

All implemented features are compatible with Cloud Foundry API v3.229.0 and follow the official CF API specification for:

- `/v3/spaces/{guid}/features` - Space feature management

For complete API documentation, see the [Cloud Foundry API v3 specification](https://v3-apidocs.cloudfoundry.org/).