# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [3.229.0] - 2026-09-03

Adds support for the CF API 3.226.0–3.229.0 delta (upstream capi-release
1.241.0–1.244.0) and refreshes every Go module dependency.

### Added

- `WithDropletCurrent()` typed list option for `Droplets().List`, sending
  `current=true` so `GET /v3/droplets` returns only each app's current
  droplet across all apps. Combinable with the other droplet filters, for
  example `WithDropletAppGUIDs(...)` to fetch current droplets for many
  apps in one request instead of one call per app. Exposed on the CLI as
  `capi apps droplets --current`. (CF API v3.229.0)

### Changed

- Route policy doc comments now match the 3.226.0 wording: the read-only
  `relationships.app`, `relationships.space`, and
  `relationships.organization` are always present with `data: null`
  unless the source references that resource type; `links` always
  carries `self` and `route` plus `app`, `space`, or `organization` when
  the source references one; the source GUID is not existence-checked at
  creation. No struct or JSON tag changes; decode tests lock in the
  documented shapes. (CF API v3.226.0)
- `Domain.EnforceRoutePolicies` and `Domain.RoutePoliciesScope` doc
  comments say "set at creation only; cannot be changed on update"
  instead of "immutable after creation", matching upstream. (CF API
  v3.226.0)
- Root `/` discovery tolerates `null` `logging`, `log_cache`, and
  `log_stream` links, which Cloud Controller returns since capi-release
  1.241.0 (cloud_controller_ng PR 5117) when those endpoints are
  unconfigured, instead of an empty `href`. This is a server behavior
  change, not a documented API change. UAA/login discovery still returns
  `ErrNoUAAOrLoginURL` when both `uaa` and `login` are null or absent.
  Covered by new tests; no behavior change.
- Dependencies: go-uaa 0.3.5 → 0.5.0, tablewriter 1.0.9 → 1.1.4, cobra
  1.9.1 → 1.10.2, viper 1.20.1 → 1.21.0, testify 1.10.0 → 1.12.1,
  golang.org/x/oauth2 0.28.0 → 0.36.0, golang.org/x/term 0.34.0 → 0.45.0,
  golang.org/x/text 0.24.0 → 0.41.0, plus the transitive graph via
  `go get -u` and `go mod tidy`. `govulncheck` reports no known
  vulnerabilities.

### Fixed

- `docs/app-features.md` listed two misspelled app feature names and a
  nonexistent `log-cache` feature. The supported features are `ssh`,
  `revisions`, `service-binding-k8s`, and `file-based-vcap-services`; the
  doc now also summarizes the upstream "Service binding files" section
  (single `vcap_services` file versus servicebinding.io directory
  projection, mutual exclusivity, size cap, and file-name rules) added
  in CF API v3.226.0.
- README and docs install snippets pointed at `v3.199.0`.

### Notes

- CF API 3.227.0 (capi-release 1.242.0) introduced asynchronous
  recursive delete for apps and service instances behind a Cloud
  Controller config flag (`temporary_enable_async_recursive_delete`).
  The v3 delete endpoints already return `202` with a job `Location`
  header, which this client follows, so no client change is required.
- CF API 3.228.0 (capi-release 1.243.0) carried no API surface changes.

## [3.225.0] - 2026-07-16

Adds support for the CF API 3.223.0–3.225.0 delta (upstream capi-release
1.238.0–1.240.0).

### Added

- Route policies resource client (`RoutePolicies()`), covering the new
  experimental identity-aware routing endpoints (CF API v3.225.0):
  `Create`, `Get`, `List`, `Update` (metadata only), and `Delete` on
  `/v3/route_policies`. Includes the `RoutePolicy` type with `Source`
  selector helpers (`RoutePolicySourceApp/Space/Organization` and
  `RoutePolicySourceAny`), typed list filters (`WithRoutePolicyGUIDs`,
  `WithRoutePolicyRouteGUIDs`, `WithRoutePolicySpaceGUIDs`,
  `WithRoutePolicySources`, `WithRoutePolicySourceGUIDs`), typed include
  options (`RoutePolicyIncludeRoute`, `RoutePolicyIncludeSource`), and
  `RoutePolicyIncludedFrom` for decoding the `included` block (routes,
  apps, spaces, organizations). New `capi route-policies` CLI command
  group (list/create/get/update/delete).
- `Domain.EnforceRoutePolicies` (`bool`) and `Domain.RoutePoliciesScope`
  (new `RoutePoliciesScope` enum: `any`, `org`, `space`) for
  identity-aware domains, settable on `DomainCreateRequest` and exposed
  via `capi domains create --enforce-route-policies
  --route-policies-scope`. Both fields are immutable after creation and
  omitted from CF responses unless enforcement is enabled. The create
  command validates the flag combinations client-side (scope requires
  enforcement and vice versa; incompatible with `--internal`), and
  `capi domains get` renders both fields when enforcement is on.
  (CF API v3.225.0)
- `RouteIncludeRoutePolicies` include option for route Get/List
  (`include=route_policies`) and `RouteIncludedResources.RoutePolicies`
  for decoding the included policies. (CF API v3.225.0)
- `Space.Suspended` (`bool`) plus `Suspended` fields on
  `SpaceCreateRequest` and `SpaceUpdateRequest`, mirroring the existing
  organization suspension surface; suspended spaces block non-admin
  writes. CLI: `capi spaces create --suspended` and a Status row in
  `capi spaces get`. (CF API v3.224.0)
- Readiness health check fields on manifest types:
  `ReadinessHealthCheckType`, `ReadinessHealthCheckHTTPEndpoint`,
  `ReadinessHealthCheckInterval`, and
  `ReadinessHealthCheckInvocationTimeout` on both `ManifestApplication`
  and `ManifestProcess`, with snake_case JSON and hyphenated YAML tags.
  (CF API v3.223.0 manifest schema)

### Notes

- Verified against CF API 3.223.0–3.225.0 release notes and docs
  (capi-release 1.238.0–1.240.0): the 3.223.0 status-code doc
  alignments (org quota apply, list roles, update user now documented
  as 200 OK) need no client change — response handling accepts any
  2xx status. The 3.224.0 lifecycle-data note (buildpacks may be
  empty, stack may be null for backfilled records) is already
  tolerated by the existing types (`Lifecycle.Data` map,
  `Droplet.Stack *string`).

## [3.222.4] - 2026-06-26

### Added

- Typed `List` entity and enum filter constructors for the endpoints that
  previously exposed only `include`/`fields`/`embed` options: apps, routes,
  spaces, roles, service instances, service plans, service offerings, service
  credential bindings, service route bindings, and processes. Every CF v3 List
  endpoint now offers the full typed filter surface (`WithAppNames`,
  `WithRouteHosts`, `WithServiceInstanceServicePlanGUIDs`, ...). New enum types
  cover their enumerated filters — `AppLifecycleType` (buildpack/cnb/docker),
  `RoleType`, `ServiceInstanceFilterType` (managed/user-provided), and
  `ServiceCredentialBindingType` (app/key) — with `available` boolean filters
  for plans and offerings. Filter and `include` options compose on a single
  `List` call; cross-resource misuse remains a compile error. No interface
  signatures changed: these endpoints already accepted variadic typed options.
- Typed `List` filter options for the remaining CF v3 collection endpoints
  that take resource-specific filters but no `include`: builds, droplets,
  packages, tasks, deployments, organizations, domains, organization quotas,
  space quotas, security groups, isolation segments, service brokers,
  buildpacks, stacks, users, audit events, and app/service usage events.
  Each endpoint exposes a sealed `XListOption` interface with `WithX...`
  constructors for its entity filters (`guids`, `app_guids`, `names`, ...)
  and typed enum constants for its enumerated filters — `BuildState`,
  `DropletState`, `PackageState`, `PackageType`, `TaskState`,
  `DeploymentState`, `DeploymentStatusValue`, `DeploymentStatusReason`,
  `BuildpackLifecycle`, and `ServiceInstanceType`. Cross-resource misuse is a
  compile error. Cross-cutting parameters (`order_by`, `label_selector`,
  `created_ats`, `updated_ats`, pagination) remain on `QueryParams`.
- Typed `include` constructors for service instances
  (`space`, `service_plan`, `service_plan.service_offering`,
  `service_plan.service_offering.service_broker`) and service offerings
  (`service_broker`), satisfying both `Get` and `List` option interfaces.
- `WithTimestampFilter(field, op, t)` helper for relational timestamp
  filters (`created_ats[gt]`, `updated_ats[lt]`, ...), validating the
  operator and emitting RFC3339 values, so callers no longer hand-build
  the bracketed query keys.
- Typed query options across resource clients, making every documented CF v3
  query parameter on these endpoints expressible:
  - Include constants for apps (`space`, `space.organization`), roles
    (`user`, `space`, `organization`), routes (`domain`, `space`,
    `space.organization`), spaces (`organization`), service plans
    (`space.organization`, `service_offering`), service credential bindings
    (`app`, `service_instance`), and service route bindings (`route`,
    `service_instance`). `Get` accepts them directly
    (`Roles().Get(ctx, guid, capi.RoleIncludeSpace)`); `List` accepts them
    after `QueryParams`. Cross-resource misuse is a compile error.
  - `fields[...]` selectors for service instances, service offerings, and
    service plans (`capi.WithServiceInstanceFields(...)` etc.), usable on
    both Get and List.
  - `embed=process_instances` for processes (`capi.ProcessEmbedInstances`).
  - Destination filters for `Routes().ListDestinations`
    (`capi.WithDestinationGUIDs`, `capi.WithDestinationAppGUIDs`).
  - `?purge=true` for `ServiceOfferings().Delete`
    (`capi.PurgeServiceOffering`).
- Typed access to `included` blocks: per-resource `XIncludedResources`
  structs on single-resource responses (`role.Included.Spaces`) and
  `capi.XIncludedFrom(list)` helpers decoding `ListResponse.Included` for
  list responses. The raw `Included` map remains available as an escape
  hatch for unmapped include types.
- `IsolationSegmentsClient.ListOrganizations`/`ListSpaces` accept
  `*QueryParams` (names/guids/paging filters per CF v3 docs).
- CLI command behavior tests: a reusable harness drives real cobra command
  trees through a client seam, captures stdout, and asserts on parsed
  arguments, client calls, rendered output, and error propagation (covering
  `sidecars get`/`delete` and `isolation-segments get`, including its
  name-lookup fallback). Previously the command tests asserted only on command
  structure. A sleep-free `CircuitBreaker` lifecycle test was added via an
  injectable clock.

### Changed

- **Breaking (interface)**: the `List` methods of 18 resource client
  interfaces (organizations, domains, builds, droplets, packages, tasks,
  deployments, buildpacks, stacks, users, service brokers, security groups,
  isolation segments, organization quotas, space quotas, audit events, and
  app/service usage events) gained a trailing variadic typed-option
  parameter. Existing `List(ctx, params)` call sites are unaffected; only
  external implementers of these interfaces must update their signatures.
- **Breaking/corrective**: several `ErrorCode*` constants carried numeric
  values that did not match the Cloud Foundry error registry. Corrected to
  the canonical CF v3 codes: `ErrorCodeNotFound` `10010`→`10000`,
  `ErrorCodeServiceUnavailable` `10001`→`10015`, `ErrorCodeInvalidRelation`
  `10020`→`1002`, `ErrorCodeMaintenanceInfo` `10012`→`390006`,
  `ErrorCodeServiceInstanceQuota` `10003`→`60005`, and
  `ErrorCodeAsyncServiceInProgress` `10001`→`60016`. The old values
  collided (for example `ServiceInstanceQuota` shared `10003` with
  `NotAuthorized`), so any code matching on them was matching the wrong
  error. `IsNotFound` now recognizes both `ErrorCodeNotFound` (10000) and
  `ErrorCodeResourceNotFound` (10010).
- **Breaking**: `CacheManager.InvalidatePattern(ctx, pattern)` is renamed to
  `InvalidateAll(ctx)`. The pattern argument was always ignored — the method
  cleared the entire cache — so the name now matches the behavior.
- **Breaking**: `CacheStats` counter fields (`Hits`, `Misses`, `Sets`,
  `Deletes`) are now accessor methods backed by `atomic.Int64`, fixing a
  data race when stats were read concurrently with cache activity.
- **Breaking (interface implementers only)**: `Get`/`List` on the clients
  listed above gained variadic option parameters, and
  `ServiceOfferings().Delete` / isolation segment list methods changed
  signature. Existing call sites compile unchanged for the variadic cases;
  external mocks implementing these interfaces must be updated, and
  isolation-segment list callers must pass params (or `nil`).

### Removed

- **Breaking**: dead exported surface removed — `SpaceWithIncludes` (the
  embedded `Space` already exposes `Included`), `PaginationHelper` /
  `NewPaginationHelper` (no callers, no methods), and the
  `ErrorCodeUniquenessError` constant (its value `10016` is CF's
  `ServiceBrokerRateLimitExceeded`, not a uniqueness error; it had no
  callers).
- **Breaking**: `ClientWithCache`, `NewClientWithCache`, and `CachedRequest`
  are removed. `ClientWithCache.Execute` always returned `ErrNotImplemented`
  via a placeholder, so the type could never serve a request and no working
  caller existed. Response caching remains available through the interceptor
  framework (`CacheInterceptor`, `ConditionalRequestInterceptor`).

### Known follow-ups

- `fields[...]` valid keys were taken from the CF v3 3.222.0 docs and
  verified at implementation time; live-CF verification is pending.
- Process `embed` response typing and `service_instances` shared-spaces
  `fields` support are deferred pending live wire capture.
- Remaining `golangci-lint` style findings are limited to `varnamelen`
  (short idiomatic names such as `i`/`v`/`w`) and `wrapcheck` at
  RoundTripper/interface boundaries — both subjective and left as-is — plus
  `goconst` hits that are overwhelmingly test fixtures. `gosec` G117 is
  excluded in `.golangci.yml` (credential structs are legitimate config);
  the CLI credential-serialization sites also carry explicit `#nosec G117`
  justifications so the standalone `make gosec` target is clean.
- Cobra command-verb and display-label string literals are intentionally left
  inline; forcing constants there harms readability for no behavioral gain.

### Fixed

- `ServiceRouteBindingsClient.Delete` now handles the synchronous delete of a
  user-provided route binding. CF v3 returns `204 No Content` (no `Location`
  header) in that case; the previous code always expected `202 + Location`
  and returned a spurious "no Location header" error. It now returns
  `(nil, nil)` on 204 and the job reference on 202, matching
  `ServiceCredentialBindingsClient.Delete`.
- `omitempty` added to optional response-struct fields that CF omits when
  null (`Buildpack.Filename`/`Stack`, `Build` staging/error/package/droplet/
  created-by refs, `Process.Command` and log-rate-limit, `Droplet.Error`,
  `Task.User`, `PackageChecksum.Value`, `FeatureFlag.CustomErrorMessage`,
  `RouteReservation.MatchingRoute`, and the space-quota sub-structs), so
  re-marshaling a response no longer emits `"field":null`.
- `WithInclude` on `QueryParams` now deduplicates repeated include values,
  matching the typed-option behavior; `WithPerPage` clamps to the CF maximum
  of 5000; and `ToValues` emits `fields[...]` and filter keys in sorted order
  so the query string is deterministic.
- `MemoryCache` operations now honor a cancelled `context.Context`, and the
  client's `New` threads the caller's context into the optional API-links
  fetch instead of using `context.Background()`.
- Library code no longer writes response-body close failures to `os.Stderr`;
  the HTTP client routes them through its configured logger and other call
  sites discard them.
- Test reliability: the wall-clock circuit-breaker test is skipped under
  `-short` (a deterministic companion covers the logic), the metrics-latency
  assertion no longer depends on scheduler timing, two always-skipped network
  tests are now real `httptest` tests, and brittle `err.Error()` substring
  assertions use `errors.Is` / `ErrorContains`.
- **Concurrency**: `MetricsCollector` and `CircuitBreaker` mutated shared map
  and counter state from per-request interceptors without synchronization
  (a concurrent map write could panic); both are now guarded by a mutex.
  `ConfigTokenManager.GetToken` updated its cached token under a read lock
  while `RefreshToken` wrote under a write lock, racing concurrent callers;
  `GetToken` now takes the write lock.
- The integration test suite (`test/integration`) did not compile — `helpers.go`
  and the test files declared different packages, so shared helpers were
  undefined — and therefore could never have run in CI. The suite now compiles
  under `-tags=integration` and carries modern `//go:build` constraints.
- `BatchTransaction` rollback was a stub that built delete operations from the
  wrong data and discarded their errors, silently leaving created resources
  behind. Rollback now deletes created resources in reverse order and reports
  what could not be reversed via `ErrRollbackIncomplete`. It is **disabled by
  default** (it is destructive); opt in with `SetRollback(true)`.
- `CacheManager` started its cleanup goroutine bound to `context.Background()`
  with no way to stop it; added `CacheManager.Close()` to cancel it.
- Pagination parsing no longer records a non-numeric `page`/`per_page` as 0,
  which could corrupt a follow-up request.
- The token-endpoint error body echoed into `ErrTokenRequestStatusFailed` is
  now bounded to 512 bytes.



- `capi isolation-segments list-spaces` worked for the first time: the
  command's internal type assertion never matched the concrete client
  (dead code since the initial commit), so it always reported
  "not supported". It now calls the typed client directly.

- **Breaking/corrective**: `OrganizationQuotaApps` and `SpaceQuotaApps` had two fields
  with wrong names that were never valid against a real CF v3 API. A live CF rejects
  create/update requests containing these fields with 422 CF-UnprocessableEntity
  "Unknown field(s): 'total_instance_memory_in_mb', 'total_app_tasks'"; reads silently
  unmarshalled them to nil because the JSON keys were absent from real CF responses.
  Both structs now use the correct CF v3 field names per the API 3.222.0 docs:
  - `TotalInstanceMemoryInMB` / `total_instance_memory_in_mb` →
    `PerProcessMemoryInMB` / `per_process_memory_in_mb`
  - `TotalAppTasks` / `total_app_tasks` →
    `PerAppTasks` / `per_app_tasks`
  All usage sites updated: `helpers.go` interfaces and builders, `org_quotas.go`,
  `space_quotas.go`, `examples/quota-management/main.go`, and client test fixtures
  which now use the real CF JSON field names.

- `DropletsClient.Delete`, `PackagesClient.Delete`, `OrganizationQuotasClient.Delete`,
  and `SpaceQuotasClient.Delete` now return `(*Job, error)` instead of bare `error`.
  CF v3 DELETE endpoints for these resources are async: 202 Accepted with an empty
  body and the job reference in the `Location` header (`/v3/jobs/{guid}`). The prior
  implementations discarded the header, leaving callers unable to poll for completion.
  Interfaces, impls, tests, CLI call sites, and examples updated in lockstep.

- `SpaceQuotaRelationships.Spaces` is now `*ToManyRelationship` with
  `omitempty` so that creating a space quota without pre-assigned spaces no
  longer serializes `"spaces":{"data":null}`, which caused CF to reject the
  request with a 422 "Relationships Spaces must be structured like this" error.
  `relationships.spaces` is optional per CF v3 API docs.

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