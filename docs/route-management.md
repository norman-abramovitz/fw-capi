# Enhanced Route Management

Enhanced route management provides advanced capabilities for route sharing and ownership transfer between spaces.

## Commands

```bash
# Share a route with one or more spaces
capi routes share ROUTE_GUID_OR_URL SPACE_GUID [SPACE_GUID...]

# Remove route sharing from a space
capi routes unshare ROUTE_GUID_OR_URL SPACE_GUID

# Transfer route ownership to a different space
capi routes transfer ROUTE_GUID_OR_URL SPACE_GUID

# List spaces that a route is shared with
capi routes list-shared ROUTE_GUID_OR_URL
```

## Examples

```bash
# Share a route with multiple spaces
capi routes share myapp.example.com space1-guid space2-guid

# Transfer route ownership
capi routes transfer myapp.example.com new-owner-space-guid

# See which spaces have access to a route
capi routes list-shared myapp.example.com

# Remove sharing from a specific space
capi routes unshare myapp.example.com space2-guid
```

## Use Cases

- **Multi-tenancy**: Share routes between development and staging spaces
- **Blue-green deployments**: Transfer route ownership during deployments
- **Access management**: Control which spaces can use specific routes

## Output Formats

All commands support three output formats:

```bash
# Human-readable table (default)
capi routes list-shared myapp.example.com

# JSON for automation and scripting
capi routes list-shared myapp.example.com --output json

# YAML for configuration management
capi routes list-shared myapp.example.com --output yaml
```

## Integration Examples

### Development Workflow

```bash
# Development setup
capi target -o my-org -s development

# Share routes between environments
capi routes share my-app-dev.example.com staging-space-guid
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
Error: route 'myapp.example.com' not found
Solution: Check route URL and ensure it exists
```

### Getting Help

```bash
# Get help for any command
capi routes --help  

# Get detailed help for subcommands
capi routes share --help
capi routes transfer --help
```

## API Compatibility

All implemented features are compatible with Cloud Foundry API v3.229.0 and follow the official CF API specification for:

- `/v3/routes/{guid}/relationships/shared_spaces` - Route sharing

For complete API documentation, see the [Cloud Foundry API v3 specification](https://v3-apidocs.cloudfoundry.org/).