package client_test

// Shared literal fixtures reused across internal/client tests. Declaring
// these once satisfies goconst's duplicate-literal check package-wide
// without touching wire-format parity assertions elsewhere, which
// intentionally keep literal JSON keys/values so a typo in a real CF v3
// field name still fails the test instead of being silently "fixed" by a
// shared constant.
const (
	// Link/JSON envelope map keys.
	testSelfKey   = "self"
	testAppKey    = "app"
	testErrorsKey = "errors"
	testCodeKey   = "code"
	testTitleKey  = "title"
	testDetailKey = "detail"

	// Common GUID fixtures.
	testAppGUID         = "app-guid"
	testSpaceGUID       = "space-guid"
	testOrgGUID         = "org-guid"
	testJobGUID         = "job-guid"
	testDropletGUID     = "droplet-guid"
	testUserGUID        = "user-guid"
	testInstanceGUID    = "instance-guid"
	testDomainGUID      = "domain-guid"
	testProcessGUID     = "process-guid"
	testPackageGUID     = "package-guid"
	testRevisionGUID    = "revision-guid"
	testQuotaGUID       = "quota-guid"
	testTargetAppGUID   = "target-app-guid"
	testBrokerGUID      = "broker-guid"
	testOfferingGUID    = "offering-guid"
	testPlanGUID        = "plan-guid"
	testBuildpackGUID   = "buildpack-guid"
	testSegmentGUID     = "segment-guid"
	testNonExistentGUID = "non-existent-guid"

	// Common short names.
	testAppNameFixture   = "test-app"
	testSpaceNameFixture = "test-space"
	testOrgNameFixture   = "test-org"
	testUserNameFixture  = "test-user"

	// Common numbered list fixtures.
	testAppGUID1      = "app-1"
	testAppGUID2      = "app-2"
	testSpaceName1    = "space-1"
	testSpaceName2    = "space-2"
	testSpaceGUID1    = "space-guid-1"
	testSpaceGUID2    = "space-guid-2"
	testOrgName1      = "org-1"
	testOrgName2      = "org-2"
	testOrgGUID1      = "org-guid-1"
	testOrgGUID2      = "org-guid-2"
	testPackageName1  = "package-1"
	testPackageName2  = "package-2"
	testDropletName2  = "droplet-2"
	testBrokerName1   = "broker-1"
	testBrokerName2   = "broker-2"
	testOfferingName1 = "offering-1"
	testOfferingName2 = "offering-2"
	testUserName1     = "user-1"

	// Process/lifecycle types.
	testWebProcessType     = "web"
	testWorkerProcessType  = "worker"
	testBuildpackLifecycle = "buildpack"

	// State literals.
	testStateStopped    = "STOPPED"
	testStateStarted    = "STARTED"
	testStateReady      = "READY"
	testStateStaged     = "STAGED"
	testStateStaging    = "STAGING"
	testStateFailed     = "FAILED"
	testStateRunning    = "RUNNING"
	testStatePending    = "PENDING"
	testStateProcessing = "PROCESSING"
	testStateDeploying  = "DEPLOYING"
	testStateDeployed   = "DEPLOYED"
	testStateFinalized  = "FINALIZED"

	// Metadata keys.
	testEnvLabelKey          = "env"
	testEnvironmentLabelKey  = "environment"
	testNoteAnnotationKey    = "note"
	testVersionAnnotationKey = "version"

	// Common query-param keys.
	testNamesParam      = "names"
	testStatesParam     = "states"
	testAppGUIDsParam   = "app_guids"
	testSpaceGUIDsParam = "space_guids"
	testOrgGUIDsParam   = "organization_guids"

	testTrueString = "true"

	testMissingAppRelationshipCase = "missing app relationship"
	testAppRelationshipRequired    = "App relationship is required"
	testSuccessfulGetCase          = "successful get"

	testHTTPProtocol = "http"
	testTCPProtocol  = "tcp"

	testRubyBuildpackName = "ruby_buildpack"

	testCreateOperation    = "create"
	testSucceededOperation = "succeeded"

	testBasicAuthType = "basic"
	testAdminUsername = "admin"
	testUserUsername  = "user"
	testPassPassword  = "pass"

	testDatabaseFixture = "database"

	testStagingLabel    = "staging"
	testProductionLabel = "production"

	testHrefAppLink = "https://api.example.org/v3/apps/app-guid"

	testBarValue = "bar"
	testValue1   = "value1"

	testEventGUID       = "event-guid"
	testMigrateTaskName = "migrate"

	testDevelopmentLabel = "development"
	testOrganizationType = "organization"
	testAPIHost          = "api"
	testAPIExampleV1URL  = "api.example.com/v1"
	testRackupCommand    = "bundle exec rackup config.ru -p $PORT"
	testDockerType       = "docker"
	testTypeKey          = "type"
	testClientIDFixture  = "client-id"
	testRollingStrategy  = "rolling"
	testVarEnvKey        = "var"
	testManagedType      = "managed"
	testMyInstanceName   = "my-instance"
	testSHA256Type       = "sha256"

	testFooKey           = "foo"
	testUpdatedValue     = "updated"
	testKeyBindingType   = "key"
	testUserProvidedType = "user-provided"
	testBitsType         = "bits"
	testTag1Value        = "tag1"

	// GUID-like "test-<resource>-guid" fixtures used by table-driven Get
	// tests; distinct from the short "<resource>-guid" fixtures above.
	testAppGUIDFixture      = "test-app-guid"
	testPackageGUIDFixture  = "test-package-guid"
	testProcessGUIDFixture  = "test-process-guid"
	testRouteGUIDFixture    = "test-route-guid"
	testBrokerGUIDFixture   = "test-broker-guid"
	testOfferingGUIDFixture = "test-offering-guid"
	testPlanGUIDFixture     = "test-plan-guid"
	testTaskGUIDFixture     = "test-task-guid"

	testBindingGUID       = "binding-guid"
	testBindingName       = "my-binding"
	testSecurityGroupGUID = "sg-guid"
	testStackGUID         = "stack-guid"

	testOrgManagerRoleType     = "organization_manager"
	testApplyManifestOperation = "app.apply_manifest"
	testFeatureFlagNameFixture = "my_feature_flag"
	testUAAOrigin              = "uaa"
	testMigrateCommand         = "rake db:migrate"
	testCFLinuxFS4Image        = "cloudfoundry/cflinuxfs4"
	testCFLinuxFS3Image        = "cloudfoundry/cflinuxfs3"
	testCFLinuxFS3Stack        = "cflinuxfs3"
	testUbuntuJammyDescription = "Ubuntu Jammy Stack"
	testOfferingNotFoundCase   = "offering not found"
	testOfferingNotFoundDetail = "Service offering not found"
	testPlanNotFoundCase       = "plan not found"
	testPlanNotFoundDetail     = "Service plan not found"
	testServiceBrokerName      = "my-service-broker"
	testServiceBrokerURL       = "https://example.service-broker.com"

	testUsernameKey      = "username"
	testPublicVisibility = "public"
)
