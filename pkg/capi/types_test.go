package capi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_JSONMarshaling(t *testing.T) {
	t.Parallel()

	resource := capi.Resource{
		GUID:      "test-guid",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Links: capi.Links{
			"self": capi.Link{
				Href: "https://api.example.org/v3/resources/test-guid",
			},
			"related": capi.Link{
				Href:   "https://api.example.org/v3/related",
				Method: http.MethodPost,
			},
		},
	}

	data, err := json.Marshal(resource)
	require.NoError(t, err)

	var decoded capi.Resource

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resource.GUID, decoded.GUID)
	assert.Equal(t, resource.CreatedAt.Unix(), decoded.CreatedAt.Unix())
	assert.Equal(t, resource.UpdatedAt.Unix(), decoded.UpdatedAt.Unix())
	assert.Equal(t, resource.Links["self"].Href, decoded.Links["self"].Href)
	assert.Equal(t, resource.Links["related"].Method, decoded.Links["related"].Method)
}

func TestMetadata_JSONMarshaling(t *testing.T) {
	t.Parallel()

	metadata := capi.Metadata{
		Labels: capi.StringMap(map[string]string{
			"environment": "production",
			"team":        "platform",
		}),
		Annotations: capi.StringMap(map[string]string{
			"version": "1.0.0",
			"owner":   "team@example.com",
		}),
	}

	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	var decoded capi.Metadata

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, metadata.Labels, decoded.Labels)
	assert.Equal(t, metadata.Annotations, decoded.Annotations)
}

func TestRelationship_JSONMarshaling(t *testing.T) {
	t.Parallel()
	t.Run("with data", func(t *testing.T) {
		t.Parallel()

		rel := capi.Relationship{
			Data: &capi.RelationshipData{
				GUID: "related-guid",
			},
		}

		data, err := json.Marshal(rel)
		require.NoError(t, err)

		var decoded capi.Relationship

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		require.NotNil(t, decoded.Data)
		assert.Equal(t, "related-guid", decoded.Data.GUID)
	})

	t.Run("without data", func(t *testing.T) {
		t.Parallel()

		rel := capi.Relationship{}

		data, err := json.Marshal(rel)
		require.NoError(t, err)

		var decoded capi.Relationship

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Nil(t, decoded.Data)
	})
}

func TestToManyRelationship_JSONMarshaling(t *testing.T) {
	t.Parallel()

	rel := capi.ToManyRelationship{
		Data: []capi.RelationshipData{
			{GUID: "guid-1"},
			{GUID: "guid-2"},
			{GUID: "guid-3"},
		},
	}

	data, err := json.Marshal(rel)
	require.NoError(t, err)

	var decoded capi.ToManyRelationship

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Data, 3)
	assert.Equal(t, "guid-1", decoded.Data[0].GUID)
	assert.Equal(t, "guid-2", decoded.Data[1].GUID)
	assert.Equal(t, "guid-3", decoded.Data[2].GUID)
}

func TestPagination_JSONMarshaling(t *testing.T) {
	t.Parallel()

	pagination := capi.Pagination{
		TotalResults: 100,
		TotalPages:   10,
		First: capi.Link{
			Href: "https://api.example.org/v3/resources?page=1",
		},
		Last: capi.Link{
			Href: "https://api.example.org/v3/resources?page=10",
		},
		Next: &capi.Link{
			Href: "https://api.example.org/v3/resources?page=2",
		},
		Previous: nil,
	}

	data, err := json.Marshal(pagination)
	require.NoError(t, err)

	var decoded capi.Pagination

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pagination.TotalResults, decoded.TotalResults)
	assert.Equal(t, pagination.TotalPages, decoded.TotalPages)
	assert.Equal(t, pagination.First.Href, decoded.First.Href)
	assert.Equal(t, pagination.Last.Href, decoded.Last.Href)
	require.NotNil(t, decoded.Next)
	assert.Equal(t, pagination.Next.Href, decoded.Next.Href)
	assert.Nil(t, decoded.Previous)
}

func TestListResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	type TestResource struct {
		capi.Resource

		Name string `json:"name"`
	}

	listResp := capi.ListResponse[TestResource]{
		Pagination: capi.Pagination{
			TotalResults: 2,
			TotalPages:   1,
			First: capi.Link{
				Href: "https://api.example.org/v3/test?page=1",
			},
			Last: capi.Link{
				Href: "https://api.example.org/v3/test?page=1",
			},
		},
		Resources: []TestResource{
			{
				Resource: capi.Resource{
					GUID: "guid-1",
				},
				Name: "test-1",
			},
			{
				Resource: capi.Resource{
					GUID: "guid-2",
				},
				Name: "test-2",
			},
		},
	}

	data, err := json.Marshal(listResp)
	require.NoError(t, err)

	var decoded capi.ListResponse[TestResource]

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, listResp.Pagination.TotalResults, decoded.Pagination.TotalResults)
	assert.Len(t, decoded.Resources, 2)
	assert.Equal(t, "guid-1", decoded.Resources[0].GUID)
	assert.Equal(t, "test-1", decoded.Resources[0].Name)
	assert.Equal(t, "guid-2", decoded.Resources[1].GUID)
	assert.Equal(t, "test-2", decoded.Resources[1].Name)
	assert.Nil(t, decoded.Included, "Included should be nil when wire payload omits the included key")
}

// TestListResponse_IncludedRoundTrip verifies that v3's `?include=...`
// joined resources survive a marshal/unmarshal round-trip on
// ListResponse[T].Included. The Included bucket is keyed by v3's
// resource-type plural names (`service_brokers`, `service_plans`,
// etc.) and holds raw JSON that callers re-decode into the concrete
// type. Mirrors the v3 wire shape:
//
//	{
//	  "pagination": {...},
//	  "resources": [...],
//	  "included": { "service_brokers": [...], "service_plans": [...] }
//	}
func TestListResponse_IncludedRoundTrip(t *testing.T) {
	t.Parallel()

	wire := []byte(`{
		"pagination": {
			"total_results": 1,
			"total_pages": 1,
			"first": {"href": "https://api.example.org/v3/service_offerings?page=1"},
			"last":  {"href": "https://api.example.org/v3/service_offerings?page=1"}
		},
		"resources": [
			{"guid": "off-1", "name": "redis"}
		],
		"included": {
			"service_brokers": [
				{"guid": "broker-1", "name": "core-broker", "url": "https://broker.example.org"}
			]
		}
	}`)

	type TestOffering struct {
		GUID string `json:"guid"`
		Name string `json:"name"`
	}

	type TestBroker struct {
		GUID string `json:"guid"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	var decoded capi.ListResponse[TestOffering]

	err := json.Unmarshal(wire, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Resources, 1)
	assert.Equal(t, "off-1", decoded.Resources[0].GUID)

	require.NotNil(t, decoded.Included)
	require.Contains(t, decoded.Included, "service_brokers")
	assert.Len(t, decoded.Included["service_brokers"], 1)

	// Late-decode the broker bucket into the concrete type.
	var brokers []TestBroker

	err = json.Unmarshal(decoded.Included["service_brokers"][0], &brokers)
	require.Error(t, err, "single raw message can't unmarshal as a slice")

	// Each entry in the bucket is one raw resource — decode entry-by-entry.
	var broker TestBroker

	err = json.Unmarshal(decoded.Included["service_brokers"][0], &broker)
	require.NoError(t, err)
	assert.Equal(t, "broker-1", broker.GUID)
	assert.Equal(t, "core-broker", broker.Name)
	assert.Equal(t, "https://broker.example.org", broker.URL)

	// Round-trip: re-marshal and ensure the included key survives.
	data, err := json.Marshal(decoded)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"included"`)
	assert.Contains(t, string(data), `"service_brokers"`)
}

// TestListResponse_IncludedOmittedWhenNil ensures the omitempty tag on
// Included drops the key from the marshalled output when the value is
// nil — preserving the existing wire shape for handlers that don't
// use `?include=`.
func TestListResponse_IncludedOmittedWhenNil(t *testing.T) {
	t.Parallel()

	resp := capi.ListResponse[capi.Resource]{
		Pagination: capi.Pagination{TotalResults: 0, TotalPages: 0},
		Resources:  []capi.Resource{},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"included"`)
}

func TestLink_MetaRoundTripsAppSSHFields(t *testing.T) {
	t.Parallel()

	// Real CF root response shape for the app_ssh link (CF 3.180.0):
	//   "app_ssh": {
	//     "href": "ssh.example.com:2222",
	//     "meta": {
	//       "host_key_fingerprint": "AAAA...",
	//       "oauth_client": "ssh-proxy"
	//     }
	//   }
	raw := []byte(`{
		"href": "ssh.example.com:2222",
		"meta": {
			"host_key_fingerprint": "AAAA-FINGERPRINT-BBBB",
			"oauth_client": "ssh-proxy"
		}
	}`)

	var link capi.Link
	require.NoError(t, json.Unmarshal(raw, &link))

	assert.Equal(t, "ssh.example.com:2222", link.Href)
	require.NotNil(t, link.Meta)
	assert.Equal(t, "AAAA-FINGERPRINT-BBBB", link.Meta["host_key_fingerprint"])
	assert.Equal(t, "ssh-proxy", link.Meta["oauth_client"])

	// Round-trip back to JSON to confirm omitempty is honored when Meta is set
	// and the field is wire-compatible.
	encoded, err := json.Marshal(link)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"meta":`)
	assert.Contains(t, string(encoded), `"host_key_fingerprint":"AAAA-FINGERPRINT-BBBB"`)
}

func TestLink_MetaOmittedWhenNil(t *testing.T) {
	t.Parallel()

	// Existing callers that don't set Meta produce wire-identical output to
	// pre-Meta versions: no `"meta":` key, no `null`.
	link := capi.Link{Href: "https://example.com/v3/apps", Method: http.MethodGet}
	encoded, err := json.Marshal(link)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), `"meta"`)
}

func TestLink_MetaCarriesAPIVersion(t *testing.T) {
	t.Parallel()

	// links.cloud_controller_v3.meta.version is the CF v3 API semver,
	// e.g. "3.180.0" — the second concrete shape Stratos consumes.
	raw := []byte(`{
		"href": "https://api.example.com/v3",
		"meta": {"version": "3.180.0"}
	}`)

	var link capi.Link
	require.NoError(t, json.Unmarshal(raw, &link))
	assert.Equal(t, "3.180.0", link.Meta["version"])
}

// TestBuildpack_OmitemptyFilenameStack verifies that nil Filename and Stack
// are omitted from the marshalled JSON (O-1, O-2).
func TestBuildpack_OmitemptyFilenameStack(t *testing.T) {
	t.Parallel()

	buildpack := capi.Buildpack{
		Resource:  capi.Resource{GUID: "bp-guid"},
		Name:      "java_buildpack",
		State:     "READY",
		Position:  1,
		Lifecycle: "buildpack",
		Enabled:   true,
		Locked:    false,
		// Filename and Stack are nil — must not appear in JSON.
	}

	data, err := json.Marshal(buildpack)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"filename"`)
	assert.NotContains(t, string(data), `"stack"`)

	// When set, the fields must appear.
	name := "java_buildpack.zip"
	stack := testStackName
	buildpack.Filename = &name
	buildpack.Stack = &stack

	data, err = json.Marshal(buildpack)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"filename":"java_buildpack.zip"`)
	assert.Contains(t, string(data), `"stack":"cflinuxfs4"`)
}

// TestPackageChecksum_OmitemptyValue verifies that nil Value is omitted (O-3).
func TestPackageChecksum_OmitemptyValue(t *testing.T) {
	t.Parallel()

	checksum := capi.PackageChecksum{Type: "sha256"}
	data, err := json.Marshal(checksum)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"value"`)

	v := testETag
	checksum.Value = &v

	data, err = json.Marshal(checksum)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"value":"abc123"`)
}

// TestDroplet_OmitemptyError verifies that nil Error is omitted from Droplet (O-4).
func TestDroplet_OmitemptyError(t *testing.T) {
	t.Parallel()

	droplet := capi.Droplet{
		State:     "STAGED",
		Lifecycle: capi.Lifecycle{Type: "buildpack", Data: map[string]any{}},
	}

	data, err := json.Marshal(droplet)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"error"`)

	msg := "staging failed"
	droplet.Error = &msg

	data, err = json.Marshal(droplet)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"error":"staging failed"`)
}

// TestBuild_OmitemptyOptionalFields verifies nil optional Build fields are omitted (O-5).
func TestBuild_OmitemptyOptionalFields(t *testing.T) {
	t.Parallel()

	build := capi.Build{
		State:             "STAGING",
		StagingMemoryInMB: 1024,
		StagingDiskInMB:   512,
		// StagingLogRateLimitBytesPerSecond, Error, Package, Droplet, CreatedBy all nil.
	}

	data, err := json.Marshal(build)
	require.NoError(t, err)

	s := string(data)
	assert.NotContains(t, s, `"staging_log_rate_limit_bytes_per_second"`)
	assert.NotContains(t, s, `"error"`)
	assert.NotContains(t, s, `"package"`)
	assert.NotContains(t, s, `"droplet"`)
	assert.NotContains(t, s, `"created_by"`)
}

// TestProcess_OmitemptyCommandAndLogRate verifies nil Command and LogRateLimit are omitted (O-6).
func TestProcess_OmitemptyCommandAndLogRate(t *testing.T) {
	t.Parallel()

	process := capi.Process{
		Type:       "web",
		Instances:  1,
		MemoryInMB: 256,
		DiskInMB:   1024,
		// Command and LogRateLimitInBytesPerSecond are nil.
	}

	data, err := json.Marshal(process)
	require.NoError(t, err)

	processJSON := string(data)
	assert.NotContains(t, processJSON, `"command"`)
	assert.NotContains(t, processJSON, `"log_rate_limit_in_bytes_per_second"`)

	cmd := "bundle exec rails server"
	rate := 1048576
	process.Command = &cmd
	process.LogRateLimitInBytesPerSecond = &rate

	data, err = json.Marshal(process)
	require.NoError(t, err)

	processJSON = string(data)
	assert.Contains(t, processJSON, `"command":"bundle exec rails server"`)
	assert.Contains(t, processJSON, `"log_rate_limit_in_bytes_per_second":1048576`)
}

// TestTask_OmitemptyUser verifies nil User is omitted from Task (O-7).
func TestTask_OmitemptyUser(t *testing.T) {
	t.Parallel()

	task := capi.Task{
		SequenceID: 1,
		Name:       "migrate",
		State:      "RUNNING",
		MemoryInMB: 256,
		DiskInMB:   1024,
		// User is nil.
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"user"`)

	u := "vcap"
	task.User = &u

	data, err = json.Marshal(task)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"user":"vcap"`)
}

// TestFeatureFlag_OmitemptyCustomErrorMessage verifies nil CustomErrorMessage is omitted (O-8).
func TestFeatureFlag_OmitemptyCustomErrorMessage(t *testing.T) {
	t.Parallel()

	featureFlag := capi.FeatureFlag{
		Name:    "app_bits_upload",
		Enabled: true,
		// CustomErrorMessage is nil.
	}

	data, err := json.Marshal(featureFlag)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"custom_error_message"`)

	msg := "Feature disabled by policy"
	featureFlag.CustomErrorMessage = &msg

	data, err = json.Marshal(featureFlag)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"custom_error_message":"Feature disabled by policy"`)
}

// TestRouteReservation_OmitemptyMatchingRoute verifies nil MatchingRoute is omitted.
func TestRouteReservation_OmitemptyMatchingRoute(t *testing.T) {
	t.Parallel()

	rr := capi.RouteReservation{
		// MatchingRoute is nil — route not reserved.
	}

	data, err := json.Marshal(rr)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"matching_route"`)
}

// TestSpaceQuota_OmitemptyAppsServicesRoutes verifies nil Apps/Services/Routes are omitted (O-9).
func TestSpaceQuota_OmitemptyAppsServicesRoutes(t *testing.T) {
	t.Parallel()

	spaceQuota := capi.SpaceQuota{
		Resource: capi.Resource{GUID: "sq-guid"},
		Name:     testPlanName,
		// Apps, Services, Routes are nil.
	}

	data, err := json.Marshal(spaceQuota)
	require.NoError(t, err)

	s := string(data)
	assert.NotContains(t, s, `"apps"`)
	assert.NotContains(t, s, `"services"`)
	assert.NotContains(t, s, `"routes"`)
}

// TestAppsQuota_OmitemptyIntFields verifies nil *int fields in AppsQuota are omitted (O-10).
func TestAppsQuota_OmitemptyIntFields(t *testing.T) {
	t.Parallel()

	appsQuota := capi.AppsQuota{
		// All nil — nothing should appear.
	}

	data, err := json.Marshal(appsQuota)
	require.NoError(t, err)

	s := string(data)
	assert.NotContains(t, s, `"total_memory_in_mb"`)
	assert.NotContains(t, s, `"per_process_memory_in_mb"`)
	assert.NotContains(t, s, `"total_instances"`)
	assert.NotContains(t, s, `"per_app_tasks"`)

	total := 2048
	appsQuota.TotalMemoryInMB = &total

	data, err = json.Marshal(appsQuota)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"total_memory_in_mb":2048`)
}

// TestServicesQuota_OmitemptyFields verifies nil *int/*bool fields in ServicesQuota are omitted (O-10).
func TestServicesQuota_OmitemptyFields(t *testing.T) {
	t.Parallel()

	servicesQuota := capi.ServicesQuota{}
	data, err := json.Marshal(servicesQuota)
	require.NoError(t, err)

	s := string(data)
	assert.NotContains(t, s, `"paid_services_allowed"`)
	assert.NotContains(t, s, `"total_service_instances"`)
	assert.NotContains(t, s, `"total_service_keys"`)

	allowed := true
	servicesQuota.PaidServicesAllowed = &allowed

	data, err = json.Marshal(servicesQuota)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"paid_services_allowed":true`)
}

// TestRoutesQuota_OmitemptyFields verifies nil *int fields in RoutesQuota are omitted (O-10).
func TestRoutesQuota_OmitemptyFields(t *testing.T) {
	t.Parallel()

	routesQuota := capi.RoutesQuota{}
	data, err := json.Marshal(routesQuota)
	require.NoError(t, err)

	s := string(data)
	assert.NotContains(t, s, `"total_routes"`)
	assert.NotContains(t, s, `"total_reserved_ports"`)

	routes := 100
	routesQuota.TotalRoutes = &routes

	data, err = json.Marshal(routesQuota)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"total_routes":100`)
}
