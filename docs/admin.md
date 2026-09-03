# Administrative Operations

Administrative operations provide platform-level management capabilities for Cloud Foundry operators.

## Commands

```bash
# Clear platform buildpack cache
capi admin clear-cache

# Get platform usage summary
capi admin usage-summary

# Get extended platform information with usage data
capi admin info
```

## Examples

```bash
# Clear buildpack cache (admin operation)
capi admin clear-cache

# View platform resource usage
capi admin usage-summary

# Get comprehensive platform information
capi admin info --output json
```

## Administrative Features

- **Cache Management**: Clear platform-wide buildpack cache
- **Usage Monitoring**: Track memory usage and instance counts
- **Platform Information**: Extended platform details with usage metrics

## Output Formats

All commands support three output formats:

```bash
# Human-readable table (default)
capi admin usage-summary

# JSON for automation and scripting
capi admin usage-summary --output json

# YAML for configuration management
capi admin info --output yaml
```

## Security Considerations

- **Permissions**: All operations require admin-level permissions
- **Authentication**: All operations require proper CF authentication
- **Audit Trail**: Operations are logged in CF audit events

## Troubleshooting

### Common Issues

**Permission Denied:**
```bash
Error: insufficient permissions to perform operation
Solution: Ensure you have admin-level permissions
```

### Getting Help

```bash
# Get help for admin commands
capi admin --help  

# Get detailed help for subcommands
capi admin clear-cache --help
capi admin usage-summary --help
```

## API Compatibility

All implemented features are compatible with Cloud Foundry API v3.229.0 and follow the official CF API specification for:

- `/v3/admin/actions/clear_buildpack_cache` - Cache management

For complete API documentation, see the [Cloud Foundry API v3 specification](https://v3-apidocs.cloudfoundry.org/).