package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestSecurityGroupsClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.SecurityGroupCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, "my-security-group", requestBody.Name)
		assert.NotNil(t, requestBody.GloballyEnabled)
		assert.True(t, requestBody.GloballyEnabled.Running)
		assert.False(t, requestBody.GloballyEnabled.Staging)
		assert.Len(t, requestBody.Rules, 2)

		now := time.Now()
		securityGroup := capi.SecurityGroup{
			Resource: capi.Resource{
				GUID:      testSecurityGroupGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:            requestBody.Name,
			GloballyEnabled: *requestBody.GloballyEnabled,
			Rules:           requestBody.Rules,
			Relationships: capi.SecurityGroupRelationships{
				RunningSpaces: capi.ToManyRelationship{
					Data: []capi.RelationshipData{},
				},
				StagingSpaces: capi.ToManyRelationship{
					Data: []capi.RelationshipData{},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(securityGroup)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	port80 := "80"
	typeICMP := 8
	codeICMP := 0
	descriptionICMP := "Allow ping"

	request := &capi.SecurityGroupCreateRequest{
		Name: "my-security-group",
		GloballyEnabled: &capi.SecurityGroupGloballyEnabled{
			Running: true,
			Staging: false,
		},
		Rules: []capi.SecurityGroupRule{
			{
				Protocol:    testTCPProtocol,
				Destination: "10.0.0.0/24",
				Ports:       &port80,
			},
			{
				Protocol:    "icmp",
				Destination: "10.0.0.0/24",
				Type:        &typeICMP,
				Code:        &codeICMP,
				Description: &descriptionICMP,
			},
		},
	}

	securityGroup, err := securityGroups.Create(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, securityGroup)
	assert.Equal(t, testSecurityGroupGUID, securityGroup.GUID)
	assert.Equal(t, "my-security-group", securityGroup.Name)
	assert.True(t, securityGroup.GloballyEnabled.Running)
	assert.Len(t, securityGroup.Rules, 2)
}

func TestSecurityGroupsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		ports := "443,80,8080"
		securityGroup := capi.SecurityGroup{
			Resource: capi.Resource{
				GUID:      testSecurityGroupGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-security-group",
			GloballyEnabled: capi.SecurityGroupGloballyEnabled{
				Running: true,
				Staging: false,
			},
			Rules: []capi.SecurityGroupRule{
				{
					Protocol:    testTCPProtocol,
					Destination: "10.10.10.0/24",
					Ports:       &ports,
				},
			},
			Relationships: capi.SecurityGroupRelationships{
				RunningSpaces: capi.ToManyRelationship{
					Data: []capi.RelationshipData{
						{GUID: testSpaceGUID1},
					},
				},
				StagingSpaces: capi.ToManyRelationship{
					Data: []capi.RelationshipData{},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(securityGroup)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	securityGroup, err := securityGroups.Get(context.Background(), testSecurityGroupGUID)
	require.NoError(t, err)
	assert.NotNil(t, securityGroup)
	assert.Equal(t, testSecurityGroupGUID, securityGroup.GUID)
	assert.Equal(t, "my-security-group", securityGroup.Name)
	assert.True(t, securityGroup.GloballyEnabled.Running)
	assert.False(t, securityGroup.GloballyEnabled.Staging)
	assert.Len(t, securityGroup.Rules, 1)
	assert.Equal(t, testTCPProtocol, securityGroup.Rules[0].Protocol)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestSecurityGroupsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "sg1,sg2", request.URL.Query().Get(testNamesParam))
		assert.Equal(t, testTrueString, request.URL.Query().Get("globally_enabled_running"))

		now := time.Now()
		response := capi.ListResponse[capi.SecurityGroup]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/security_groups?page=1"},
				Last:         capi.Link{Href: "/v3/security_groups?page=1"},
			},
			Resources: []capi.SecurityGroup{
				{
					Resource: capi.Resource{
						GUID:      "sg-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "sg1",
					GloballyEnabled: capi.SecurityGroupGloballyEnabled{
						Running: true,
						Staging: false,
					},
					Rules: []capi.SecurityGroupRule{},
				},
				{
					Resource: capi.Resource{
						GUID:      "sg-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "sg2",
					GloballyEnabled: capi.SecurityGroupGloballyEnabled{
						Running: true,
						Staging: true,
					},
					Rules: []capi.SecurityGroupRule{},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	params := &capi.QueryParams{
		Filters: map[string][]string{
			testNamesParam:             {"sg1", "sg2"},
			"globally_enabled_running": {testTrueString},
		},
	}

	list, err := securityGroups.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "sg1", list.Resources[0].Name)
	assert.Equal(t, "sg2", list.Resources[1].Name)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestSecurityGroupsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.SecurityGroupUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.NotNil(t, requestBody.Name)
		assert.Equal(t, "updated-sg", *requestBody.Name)
		assert.NotNil(t, requestBody.GloballyEnabled)
		assert.False(t, requestBody.GloballyEnabled.Running)
		assert.True(t, requestBody.GloballyEnabled.Staging)

		now := time.Now()
		securityGroup := capi.SecurityGroup{
			Resource: capi.Resource{
				GUID:      testSecurityGroupGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:            *requestBody.Name,
			GloballyEnabled: *requestBody.GloballyEnabled,
			Rules:           requestBody.Rules,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(securityGroup)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	name := "updated-sg"
	ports := "443"
	request := &capi.SecurityGroupUpdateRequest{
		Name: &name,
		GloballyEnabled: &capi.SecurityGroupGloballyEnabled{
			Running: false,
			Staging: true,
		},
		Rules: []capi.SecurityGroupRule{
			{
				Protocol:    testTCPProtocol,
				Destination: "192.168.0.0/16",
				Ports:       &ports,
			},
		},
	}

	securityGroup, err := securityGroups.Update(context.Background(), testSecurityGroupGUID, request)
	require.NoError(t, err)
	assert.NotNil(t, securityGroup)
	assert.Equal(t, testSecurityGroupGUID, securityGroup.GUID)
	assert.Equal(t, "updated-sg", securityGroup.Name)
	assert.False(t, securityGroup.GloballyEnabled.Running)
	assert.True(t, securityGroup.GloballyEnabled.Staging)
}

func TestSecurityGroupsClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/security_groups/{guid} is async: 202 Accepted, empty
	// body, Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	job, err := securityGroups.Delete(context.Background(), testSecurityGroupGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestSecurityGroupsClient_BindRunningSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid/relationships/running_spaces", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.SecurityGroupBindRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Len(t, requestBody.Data, 2)
		assert.Equal(t, testSpaceGUID1, requestBody.Data[0].GUID)
		assert.Equal(t, testSpaceGUID2, requestBody.Data[1].GUID)

		response := capi.ToManyRelationship(requestBody)

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	relationship, err := securityGroups.BindRunningSpaces(context.Background(), testSecurityGroupGUID, []string{testSpaceGUID1, testSpaceGUID2})
	require.NoError(t, err)
	assert.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 2)
}

func TestSecurityGroupsClient_UnbindRunningSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid/relationships/running_spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	err := securityGroups.UnbindRunningSpace(context.Background(), testSecurityGroupGUID, testSpaceGUID)
	require.NoError(t, err)
}

func TestSecurityGroupsClient_BindStagingSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid/relationships/staging_spaces", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.SecurityGroupBindRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Len(t, requestBody.Data, 1)
		assert.Equal(t, testSpaceGUID1, requestBody.Data[0].GUID)

		response := capi.ToManyRelationship(requestBody)

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	relationship, err := securityGroups.BindStagingSpaces(context.Background(), testSecurityGroupGUID, []string{testSpaceGUID1})
	require.NoError(t, err)
	assert.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 1)
	assert.Equal(t, testSpaceGUID1, relationship.Data[0].GUID)
}

func TestSecurityGroupsClient_UnbindStagingSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/security_groups/sg-guid/relationships/staging_spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	securityGroups := NewSecurityGroupsClient(httpClient)

	err := securityGroups.UnbindStagingSpace(context.Background(), testSecurityGroupGUID, testSpaceGUID)
	require.NoError(t, err)
}
