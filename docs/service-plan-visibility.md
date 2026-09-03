# Service Plan Visibility Management

Service plan visibility controls which organizations and spaces can access specific service plans.

## Commands

```bash
# Get current visibility settings for a service plan
capi services plans visibility get SERVICE_PLAN_NAME_OR_GUID

# Update visibility settings
capi services plans visibility update SERVICE_PLAN_NAME_OR_GUID --type TYPE [--orgs ORG1,ORG2]

# Apply visibility settings  
capi services plans visibility apply SERVICE_PLAN_NAME_OR_GUID --type TYPE [--orgs ORG1,ORG2]

# Remove organization from plan visibility
capi services plans visibility remove-org SERVICE_PLAN_NAME_OR_GUID ORG_GUID
```

## Visibility Types

- `public` - Available to all organizations
- `admin` - Available only to admin users
- `organization` - Available to specific organizations (use `--orgs` flag)
- `space` - Available to specific spaces

## Examples

```bash
# Make a service plan public
capi services plans visibility update postgres-small --type public

# Restrict plan to specific organizations
capi services plans visibility update postgres-large --type organization --orgs org1-guid,org2-guid

# Check current visibility
capi services plans visibility get postgres-small

# Remove organization access
capi services plans visibility remove-org postgres-large org3-guid
```

## Output Formats

All commands support three output formats:

```bash
# Human-readable table (default)
capi services plans visibility get postgres-small

# JSON for automation and scripting
capi services plans visibility get postgres-small --output json

# YAML for configuration management
capi services plans visibility get postgres-small --output yaml
```

## Integration Examples

### CI/CD Pipeline Integration

```bash
#!/bin/bash
# Configure service visibility
capi services plans visibility update postgres-prod --type organization --orgs prod-org-guid
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
Error: service plan 'postgres-small' not found
Solution: Check service plan name and ensure it exists
```

### Getting Help

```bash
# Get help for any command
capi services plans visibility --help  

# Get detailed help for subcommands
capi services plans visibility update --help
```

## API Compatibility

All implemented features are compatible with Cloud Foundry API v3.229.0 and follow the official CF API specification for:

- `/v3/service_plans/{guid}/visibility` - Service plan visibility

For complete API documentation, see the [Cloud Foundry API v3 specification](https://v3-apidocs.cloudfoundry.org/).