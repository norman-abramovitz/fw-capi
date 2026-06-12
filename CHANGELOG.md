# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `DropletsClient.Delete`, `PackagesClient.Delete`, `OrganizationQuotasClient.Delete`,
  and `SpaceQuotasClient.Delete` now return `(*Job, error)` instead of bare `error`.
  CF v3 DELETE endpoints for these resources are async: 202 Accepted with an empty
  body and the job reference in the `Location` header (`/v3/jobs/{guid}`). The prior
  implementations discarded the header, leaving callers unable to poll for completion.
  Interfaces, impls, tests, CLI call sites, and examples updated in lockstep.

## [3.216.4] - 2026-04-24

### Removed

- NATS JetStream KV cache backend (`pkg/capi/cache_nats.go`, `CacheTypeNATS`,
  `CacheConfig.NATS`, `CacheBuilder.WithNATSConfig`). The backend was opt-in,
  untested, and unused by known consumers; its presence pulled `nats.go`,
  `nkeys`, and `nuid` into every dependent build. Memory and no-op cache
  backends remain. Patch bump (additive removal of an optional, dead
  feature) per maintainer direction; downstream code that explicitly
  selected `CacheTypeNATS` must migrate to memory or no-op.

### Added
- Initial release of Cloud Foundry API v3 client library
- Complete CF API v3 resource coverage
- CLI tool with full CF operations support
- Multiple authentication methods (username/password, client credentials, access token)
- Automatic UAA endpoint discovery
- Comprehensive error handling with CF API error types
- Pagination support with automatic page handling
- Query parameter builder with filtering and includes
- Pluggable caching backends (memory, Redis, NATS)
- Rate limiting with configurable policies
- Interceptors for request/response customization
- Batch operations support
- Streaming API for large datasets
- Rich metadata support (labels and annotations)
- Integration and unit test suites
- Comprehensive documentation and examples

### Security
- Secure credential handling
- TLS certificate validation (configurable)
- OAuth2 token management with automatic refresh

## [3.216.0] - 2026-03-31

### Changed

- Tracks Cloud Foundry API v3.216.0 (no API-level structural changes from v3.215.0)

- Server-side behavioral changes:

  - App names now have max length validation enforced by the API

  - Default `max_service_credential_bindings_per_app_service_instance` is now 5

## [3.215.0] - 2026-03-31

### Changed

- App features `k8s-service-bindings` and `file-based-service-bindings` graduated
  from experimental to generally available (CF API v3.215.0)

## [3.214.0] - 2026-03-31

### Changed

- Tracks Cloud Foundry API v3.214.0 (no API-level changes from v3.213.0)

## [3.213.0] - 2026-03-31

### Changed

- Tracks Cloud Foundry API v3.213.0 (no API-level changes from v3.212.0)

## [3.212.0] - 2026-02-19

### Changed

- Tracks Cloud Foundry API v3.212.0 (no API-level changes from v3.211.0)

## [3.211.0] - 2026-02-19

### Added

- `ProcessInstance` struct: represents a process instance from the new
  `GET /v3/processes/:guid/process_instances` endpoint. Fields:

  - `Index` (`int`): zero-based index of the instance

  - `State` (`string`): instance state (`RUNNING`, `CRASHED`, `STARTING`, `STOPPING`, `DOWN`)

  - `Since` (`int`): seconds since the instance entered its current state

  (CF API v3.211.0)

- `ProcessesClient.ListInstances` method: calls `GET /v3/processes/:guid/process_instances`
  and returns `*ListResponse[ProcessInstance]`. (CF API v3.211.0)

- `Stack.StateReason` field (`string`, optional): plain text describing the stack state change.
  Also available on `StackCreateRequest.StateReason` (`string`) and
  `StackUpdateRequest.StateReason` (`*string`). (CF API v3.211.0)

- `embed` query parameter for listing Processes: comma-delimited list of resources to embed
  (valid value: `process_instances`). Pass via `QueryParams.Filters["embed"]`. (CF API v3.211.0)

## [3.210.0] - 2026-02-19

### Added

- `Route.Options` field (`*RouteOptions`, optional): load-balancing options for a route.
  The `RouteOptions` struct contains:

  - `Loadbalancing` (`*string`): load-balancer algorithm — `"round-robin"`, `"least-connection"`,
    or `"hash"`

  - `HashHeader` (`*string`): HTTP header name to hash for routing (e.g., `"X-User-ID"`,
    `"Cookie"`); required when `Loadbalancing` is `"hash"`

  - `HashBalance` (`*string`): weight factor for load balancing (`"1.1"` to `"10"`, or `"0"` to
    disable); optional when `Loadbalancing` is `"hash"`

  Also available on `RouteCreateRequest.Options` and `RouteUpdateRequest.Options`. (CF API v3.210.0)

- `Stack.State` field (`string`): the state of the stack. Valid values: `ACTIVE`, `RESTRICTED`,
  `DEPRECATED`, `DISABLED`. Also available on `StackCreateRequest.State` (`string`) and
  `StackUpdateRequest.State` (`*string`). (CF API v3.210.0)

- `broker_catalog_ids` query parameter for listing Service Offerings: comma-delimited list of IDs
  provided by the service broker to filter by. Pass via `QueryParams.Filters["broker_catalog_ids"]`.
  (CF API v3.210.0)

## [3.209.0] - 2026-02-19

### Added

- `ServiceInstance.BrokerProvidedMetadata` field (`*ServiceInstanceBrokerProvidedMetadata`, optional):
  metadata provided by the service broker about managed service instances. Contains:

  - `Attributes` (`map[string]interface{}`): broker-specific key-value pairs that may imply
    behavior changes by the platform

  - `Labels` (`map[string]interface{}`): broker-specified key-value pairs for attributes that
    do not directly imply behavior changes

  Only shown when `Type` is `"managed"`. (CF API v3.209.0)

## [3.208.0] - 2026-02-19

### Changed

- Tracks Cloud Foundry API v3.208.0 (no API-level changes from v3.207.0)

## [3.207.0] - 2026-02-19

### Changed

- Tracks Cloud Foundry API v3.207.0 (no API-level changes from v3.206.0)

## [3.206.0] - 2026-02-19

### Changed

- Tracks Cloud Foundry API v3.206.0 (no API-level changes from v3.205.0)

## [3.205.0] - 2026-02-18

### Added

- `ServiceCredentialBindingCreateRequest.Strategy` field (`string`, optional): sets the binding
  creation strategy. Valid values are `single` (default) and `multiple` (experimental).
  Only applicable when `Type` is `"app"`. (CF API v3.205.0)

## [3.204.0] - 2026-02-18

### Changed

- Tracks Cloud Foundry API v3.204.0 (no API-level changes from v3.203.0)

## [3.203.0] - 2026-02-18

### Changed

- Tracks Cloud Foundry API v3.203.0 (no API-level changes from v3.202.0)

## [1.0.0] - 2024-01-15

### Added

#### Core Client Library
- **Complete CF API v3 Coverage**: Full support for all Cloud Foundry API v3 resources
  - Organizations, Spaces, Applications, Services
  - Users, Roles, Routes, Domains
  - Buildpacks, Stacks, Feature Flags
  - Jobs, Tasks, Processes, Deployments
  - Service Instances, Service Bindings, Service Brokers
  - And many more resources

#### Authentication & Security
- **Multiple Authentication Methods**:
  - Username/Password with OAuth2 Resource Owner Password Credentials
  - Client Credentials for service-to-service authentication  
  - Access Token authentication
  - Refresh Token with automatic renewal
- **Automatic UAA Discovery**: Client automatically discovers UAA endpoint from CF API
- **Secure Token Management**: Automatic token refresh and secure storage
- **TLS Support**: Configurable TLS verification for development environments

#### Advanced Features
- **Intelligent Pagination**: 
  - Manual pagination with full control
  - Automatic pagination with callback functions
  - Memory-efficient streaming for large datasets
- **Rich Query Support**:
  - Type-safe query parameter builder
  - Filtering by multiple criteria
  - Include relationships in responses
  - Sorting and ordering
- **Comprehensive Error Handling**:
  - Structured CF API error types
  - Rich error context and details
  - Retry logic with exponential backoff
- **Performance Optimizations**:
  - HTTP connection pooling
  - Configurable timeouts and rate limiting
  - Request/response compression
  - Pluggable caching backends

#### Caching System
- **Memory Cache**: High-performance in-memory caching with LRU eviction
- **Redis Cache**: Distributed caching with Redis backend
- **NATS Cache**: Event-driven caching with NATS messaging
- **Custom Cache Backends**: Pluggable architecture for custom implementations
- **TTL Management**: Configurable time-to-live for cached responses

#### CLI Tool
- **Full-Featured CLI**: Complete command-line interface for CF operations
- **Interactive Authentication**: Secure credential prompting
- **Multiple Output Formats**: Table, JSON, and YAML output
- **Configuration Management**: Persistent configuration with multiple targets
- **Batch Operations**: Efficient bulk operations
- **Rich Command Set**:
  - Organization and space management
  - Application lifecycle operations
  - Service management and bindings
  - User and role administration
  - Route and domain management

#### Developer Experience
- **Type Safety**: Full Go type definitions for all CF API resources
- **Rich Metadata**: Support for labels and annotations on all resources  
- **Context Support**: Proper context handling for cancellation and timeouts
- **Interceptors**: Request/response interceptors for custom processing
- **Comprehensive Testing**: Unit tests, integration tests, and mocks
- **Extensive Documentation**: API docs, examples, and guides

#### Resource Operations

##### Organizations
- List organizations with filtering and pagination
- Create organizations with metadata and quotas
- Update organization properties and metadata
- Delete organizations with job tracking
- Manage organization users and roles
- Organization quota management

##### Spaces  
- List spaces with organization filtering
- Create spaces with isolation segments
- Update space properties and metadata
- Delete spaces and associated resources
- Space quota management
- Space role assignments

##### Applications
- Complete application lifecycle management
- Start, stop, restart applications
- Scale applications (instances, memory, disk)
- Environment variable management
- Application stats and process information
- Deployment and rollback operations
- Log streaming and events

##### Services
- Service offering and plan discovery
- Managed service instance creation
- User-provided service instances
- Service binding management
- Service key generation
- Service broker operations
- Service usage events

##### Users and Roles
- User management across organizations and spaces
- Role assignments and permissions
- User invitation and authentication
- Service account management

##### Routes and Domains
- Route creation and mapping
- Domain management (shared and private)
- SSL certificate handling
- Route service bindings

##### Additional Resources
- Buildpack management
- Stack operations
- Feature flag configuration
- Security group management
- Isolation segment handling

#### Examples and Documentation
- **Comprehensive Examples**: Over 20 example programs covering all major use cases
- **API Documentation**: Detailed API reference with code samples
- **CLI Documentation**: Complete command reference and usage examples
- **Integration Guides**: Step-by-step integration instructions
- **Best Practices**: Performance and security guidelines
- **Troubleshooting**: Common issues and solutions

### Changed
- N/A (Initial release)

### Deprecated
- N/A (Initial release)

### Removed
- N/A (Initial release)

### Fixed
- N/A (Initial release)

### Security
- Implemented secure credential handling
- Added TLS certificate validation with bypass option for development
- OAuth2 token security with automatic refresh
- Secure configuration file permissions (0600)

## Development History

### Phase 1: Project Foundation (2023-10-01 - 2023-10-15)
- Project structure and basic setup
- Go module initialization
- Core package architecture
- Basic HTTP client implementation

### Phase 2-15: Core API Implementation (2023-10-16 - 2023-11-15)
- Implementation of all CF API v3 resource clients
- HTTP client with authentication
- Error handling and response parsing
- Basic query parameter support

### Phase 16-25: Advanced Features (2023-11-16 - 2023-12-15)
- Pagination handling
- Advanced query parameters
- Caching system implementation  
- Rate limiting and performance optimizations
- Comprehensive error handling

### Phase 26-30: Testing and Polish (2023-12-16 - 2024-01-10)
- Unit test suite
- Integration test framework
- Mock implementations
- Performance testing
- Documentation improvements

### Phase 31: CLI Implementation (2024-01-11 - 2024-01-14)
- Command-line interface development
- Authentication flow implementation
- Output formatting and configuration
- Command structure and help system

### Phase 32: Documentation and Examples (2024-01-15)
- Comprehensive documentation
- Example programs for all major features
- API reference documentation
- Contributing guidelines
- Release preparation

## Migration Guide

### From CF CLI to CAPI Client

The CAPI client provides a programmatic interface that complements the CF CLI:

```bash
# CF CLI
cf orgs

# CAPI CLI  
capi orgs list
```

```go
// CAPI Client Library
orgs, err := client.Organizations().List(ctx, nil)
```

### Authentication Migration

```bash
# CF CLI
cf login -a api.example.com -u user -p pass

# CAPI CLI
capi login -a api.example.com -u user -p pass
```

```go
// CAPI Client Library
client, err := cfclient.NewWithPassword(
    "https://api.example.com", 
    "user", 
    "pass",
)
```

## Roadmap

### Upcoming Features (v1.1.0)
- GraphQL API support
- Enhanced streaming capabilities  
- Advanced caching strategies
- Performance monitoring integration
- Additional authentication methods

### Future Releases
- v1.2.0: Enhanced CLI features and plugins
- v1.3.0: Advanced automation and workflow support
- v2.0.0: Next-generation API support

## Support

- **Documentation**: https://pkg.go.dev/github.com/fivetwenty-io/capi-client
- **Issues**: https://github.com/fivetwenty-io/capi-client/issues
- **Discussions**: https://github.com/fivetwenty-io/capi-client/discussions
- **Examples**: https://github.com/fivetwenty-io/capi-client/tree/main/examples

## Contributors

- Initial implementation by the FiveTwenty.io Platform Team
- Special thanks to the Cloud Foundry community for API specifications
- Contributors welcome! See [CONTRIBUTING.md](CONTRIBUTING.md)

---

**Note**: This project follows [Semantic Versioning](https://semver.org/). 
For the full commit history and detailed changes, see the [GitHub repository](https://github.com/fivetwenty-io/capi-client).