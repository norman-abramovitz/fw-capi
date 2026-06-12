package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServicePlansClient_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "successful get",
			guid:         "test-plan-guid",
			expectedPath: "/v3/service_plans/test-plan-guid",
			statusCode:   http.StatusOK,
			response: capi.ServicePlan{
				Resource: capi.Resource{
					GUID:      "test-plan-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
							Href: "https://api.example.org/v3/service_plans/test-plan-guid",
						},
						"service_offering": capi.Link{
							Href: "https://api.example.org/v3/service_offerings/offering-guid",
						},
						"visibility": capi.Link{
							Href: "https://api.example.org/v3/service_plans/test-plan-guid/visibility",
						},
					},
				},
				Name:           "my_big_service_plan",
				Description:    "Big",
				Available:      true,
				VisibilityType: "public",
				Free:           false,
				Costs: []capi.ServicePlanCost{
					{
						Currency: "USD",
						Amount:   199.99,
						Unit:     "Monthly",
					},
				},
				MaintenanceInfo: &capi.ServicePlanMaintenance{
					Version:     "1.0.0+dev4",
					Description: "Database version 7.8.0",
				},
				BrokerCatalog: capi.ServicePlanCatalog{
					ID: "db730a8c-11e5-11ea-838a-0f4fff3b1cfb",
					Metadata: map[string]interface{}{
						"custom-key": "custom-information",
					},
					Features: capi.ServicePlanCatalogFeatures{
						PlanUpdateable: true,
						Bindable:       true,
					},
				},
				Schemas: capi.ServicePlanSchemas{
					ServiceInstance: capi.ServiceInstanceSchema{
						Create: capi.SchemaDefinition{
							Parameters: map[string]interface{}{
								"$schema": "http://json-schema.org/draft-04/schema#",
								"type":    "object",
							},
						},
						Update: capi.SchemaDefinition{
							Parameters: map[string]interface{}{},
						},
					},
					ServiceBinding: capi.ServiceBindingSchema{
						Create: capi.SchemaDefinition{
							Parameters: map[string]interface{}{},
						},
					},
				},
				Relationships: capi.ServicePlanRelationships{
					ServiceOffering: capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "offering-guid",
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"type": "database",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "plan not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_plans/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Service plan not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "GET", request.Method)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			plan, err := client.ServicePlans().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, plan)
			} else {
				require.NoError(t, err)
				require.NotNil(t, plan)
				assert.Equal(t, testCase.guid, plan.GUID)
				assert.Equal(t, "my_big_service_plan", plan.Name)
				assert.Equal(t, "public", plan.VisibilityType)
				assert.True(t, plan.Available)
				assert.False(t, plan.Free)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for detailed testing
func TestServicePlansClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_plans", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if names := query.Get("names"); names != "" {
			assert.Equal(t, "plan1,plan2", names)
		}

		if offeringGuids := query.Get("service_offering_guids"); offeringGuids != "" {
			assert.Equal(t, "offering-1,offering-2", offeringGuids)
		}

		if available := query.Get("available"); available != "" {
			assert.Equal(t, "true", available)
		}

		response := capi.ListResponse[capi.ServicePlan]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/service_plans?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/service_plans?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.ServicePlan{
				{
					Resource: capi.Resource{
						GUID:      "plan-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:           "small_plan",
					Description:    "Small database",
					Available:      true,
					VisibilityType: "public",
					Free:           true,
					Costs:          []capi.ServicePlanCost{},
					BrokerCatalog: capi.ServicePlanCatalog{
						ID: "catalog-id-1",
						Features: capi.ServicePlanCatalogFeatures{
							Bindable: true,
						},
					},
					Schemas: capi.ServicePlanSchemas{
						ServiceInstance: capi.ServiceInstanceSchema{
							Create: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
							Update: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
						},
						ServiceBinding: capi.ServiceBindingSchema{
							Create: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
						},
					},
					Relationships: capi.ServicePlanRelationships{
						ServiceOffering: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "offering-1",
							},
						},
					},
				},
				{
					Resource: capi.Resource{
						GUID:      "plan-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:           "large_plan",
					Description:    "Large database",
					Available:      false,
					VisibilityType: "organization",
					Free:           false,
					Costs: []capi.ServicePlanCost{
						{
							Amount:   99.99,
							Currency: "USD",
							Unit:     "Monthly",
						},
					},
					BrokerCatalog: capi.ServicePlanCatalog{
						ID: "catalog-id-2",
						Features: capi.ServicePlanCatalogFeatures{
							Bindable:       true,
							PlanUpdateable: true,
						},
					},
					Schemas: capi.ServicePlanSchemas{
						ServiceInstance: capi.ServiceInstanceSchema{
							Create: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
							Update: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
						},
						ServiceBinding: capi.ServiceBindingSchema{
							Create: capi.SchemaDefinition{
								Parameters: map[string]interface{}{},
							},
						},
					},
					Relationships: capi.ServicePlanRelationships{
						ServiceOffering: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "offering-2",
							},
						},
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	// Test without filters
	result, err := client.ServicePlans().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "plan-1", result.Resources[0].GUID)
	assert.Equal(t, "small_plan", result.Resources[0].Name)
	assert.True(t, result.Resources[0].Free)
	assert.Equal(t, "plan-2", result.Resources[1].GUID)
	assert.Equal(t, "large_plan", result.Resources[1].Name)
	assert.False(t, result.Resources[1].Free)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"names":                  {"plan1", "plan2"},
			"service_offering_guids": {"offering-1", "offering-2"},
			"available":              {"true"},
		},
	}
	result, err = client.ServicePlans().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

//nolint:funlen // Test functions can be longer for detailed testing
func TestServicePlansClient_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		request      *capi.ServicePlanUpdateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "successful update",
			guid:         "test-plan-guid",
			expectedPath: "/v3/service_plans/test-plan-guid",
			statusCode:   http.StatusOK,
			request: &capi.ServicePlanUpdateRequest{
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
					Annotations: map[string]string{
						"note": "Updated plan",
					},
				},
			},
			response: capi.ServicePlan{
				Resource: capi.Resource{
					GUID:      "test-plan-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:           "my_service_plan",
				Description:    "A service plan",
				Available:      true,
				VisibilityType: "public",
				Free:           false,
				BrokerCatalog: capi.ServicePlanCatalog{
					ID: "catalog-id",
					Features: capi.ServicePlanCatalogFeatures{
						Bindable: true,
					},
				},
				Schemas: capi.ServicePlanSchemas{
					ServiceInstance: capi.ServiceInstanceSchema{
						Create: capi.SchemaDefinition{
							Parameters: map[string]interface{}{},
						},
						Update: capi.SchemaDefinition{
							Parameters: map[string]interface{}{},
						},
					},
					ServiceBinding: capi.ServiceBindingSchema{
						Create: capi.SchemaDefinition{
							Parameters: map[string]interface{}{},
						},
					},
				},
				Relationships: capi.ServicePlanRelationships{
					ServiceOffering: capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "offering-guid",
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
					Annotations: map[string]string{
						"note": "Updated plan",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "plan not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_plans/non-existent-guid",
			statusCode:   http.StatusNotFound,
			request: &capi.ServicePlanUpdateRequest{
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"test": "value",
					},
				},
			},
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Service plan not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "PATCH", request.Method)

				var requestBody capi.ServicePlanUpdateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			plan, err := client.ServicePlans().Update(context.Background(), testCase.guid, testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, plan)
			} else {
				require.NoError(t, err)
				require.NotNil(t, plan)
				assert.Equal(t, testCase.guid, plan.GUID)
				assert.Equal(t, "my_service_plan", plan.Name)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for detailed testing
func TestServicePlansClient_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
		response     interface{}
	}{
		{
			name:         "successful delete",
			guid:         "test-plan-guid",
			expectedPath: "/v3/service_plans/test-plan-guid",
			statusCode:   http.StatusNoContent,
			wantErr:      false,
		},
		{
			name:         "plan not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_plans/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Service plan not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
		{
			name:         "plan has service instances",
			guid:         "plan-with-instances",
			expectedPath: "/v3/service_plans/plan-with-instances",
			statusCode:   http.StatusUnprocessableEntity,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10008,
						"title":  "CF-UnprocessableEntity",
						"detail": "Service plan has service instances",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-UnprocessableEntity",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "DELETE", request.Method)

				if testCase.response != nil {
					writer.Header().Set("Content-Type", "application/json")
				}

				writer.WriteHeader(testCase.statusCode)

				if testCase.response != nil {
					_ = json.NewEncoder(writer).Encode(testCase.response)
				}
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			err = client.ServicePlans().Delete(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServicePlansClient_GetVisibility(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_plans/test-plan-guid/visibility", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		response := capi.ServicePlanVisibility{
			Type: "organization",
			Organizations: []capi.ServicePlanVisibilityOrg{
				{
					GUID: "org-1",
					Name: "Organization One",
				},
				{
					GUID: "org-2",
					Name: "Organization Two",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	visibility, err := client.ServicePlans().GetVisibility(context.Background(), "test-plan-guid")
	require.NoError(t, err)
	require.NotNil(t, visibility)
	assert.Equal(t, "organization", visibility.Type)
	assert.Len(t, visibility.Organizations, 2)
	assert.Equal(t, "org-1", visibility.Organizations[0].GUID)
}

func TestServicePlansClient_UpdateVisibility(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_plans/test-plan-guid/visibility", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.ServicePlanVisibilityUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "organization", requestBody.Type)
		assert.Contains(t, requestBody.Organizations, "org-1")

		response := capi.ServicePlanVisibility{
			Type: "organization",
			Organizations: []capi.ServicePlanVisibilityOrg{
				{
					GUID: "org-1",
					Name: "Organization One",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.ServicePlanVisibilityUpdateRequest{
		Type:          "organization",
		Organizations: []string{"org-1"},
	}

	visibility, err := client.ServicePlans().UpdateVisibility(context.Background(), "test-plan-guid", request)
	require.NoError(t, err)
	require.NotNil(t, visibility)
	assert.Equal(t, "organization", visibility.Type)
	assert.Len(t, visibility.Organizations, 1)
}

func TestServicePlansClient_RemoveOrgFromVisibility(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_plans/test-plan-guid/visibility/org-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.ServicePlans().RemoveOrgFromVisibility(context.Background(), "test-plan-guid", "org-guid")
	require.NoError(t, err)
}

func TestServicePlansClient_GetWithIncludeAndFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "service_offering", request.URL.Query().Get("include"))
		assert.Equal(t, "name", request.URL.Query().Get("fields[service_offering.service_broker]"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "plan-guid",
		  "included": {
		    "service_offerings": [{"guid": "offering-1"}],
		    "service_brokers": [{"name": "broker-1"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	plans := NewServicePlansClient(httpClient)

	plan, err := plans.Get(context.Background(), "plan-guid",
		capi.ServicePlanIncludeServiceOffering,
		capi.WithServicePlanFields(capi.ServicePlanFieldsServiceOfferingServiceBroker, "name"))
	require.NoError(t, err)
	require.NotNil(t, plan.Included)
	assert.Equal(t, "offering-1", plan.Included.ServiceOfferings[0].GUID)
}
