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
func TestServiceOfferingsClient_Get(t *testing.T) {
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
			guid:         "test-offering-guid",
			expectedPath: "/v3/service_offerings/test-offering-guid",
			statusCode:   http.StatusOK,
			response: capi.ServiceOffering{
				Resource: capi.Resource{
					GUID:      "test-offering-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
							Href: "https://api.example.org/v3/service_offerings/test-offering-guid",
						},
						"service_plans": capi.Link{
							Href: "https://api.example.org/v3/service_plans?service_offering_guids=test-offering-guid",
						},
						"service_broker": capi.Link{
							Href: "https://api.example.org/v3/service_brokers/broker-guid",
						},
					},
				},
				Name:             "my_service_offering",
				Description:      "Provides my service",
				Available:        true,
				Tags:             []string{"relational", "caching"},
				Requires:         []string{},
				Shareable:        true,
				DocumentationURL: StringPtr("https://some-documentation-link.io"),
				BrokerCatalog: capi.ServiceOfferingCatalog{
					ID: "db730a8c-11e5-11ea-838a-0f4fff3b1cfb",
					Metadata: map[string]interface{}{
						"shareable": true,
					},
					Features: capi.ServiceOfferingCatalogFeatures{
						PlanUpdateable:       true,
						Bindable:             true,
						InstancesRetrievable: true,
						BindingsRetrievable:  true,
						AllowContextUpdates:  false,
					},
				},
				Relationships: capi.ServiceOfferingRelationships{
					ServiceBroker: capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "broker-guid",
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
			name:         "offering not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_offerings/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Service offering not found",
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

			offering, err := client.ServiceOfferings().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, offering)
			} else {
				require.NoError(t, err)
				require.NotNil(t, offering)
				assert.Equal(t, testCase.guid, offering.GUID)
				assert.Equal(t, "my_service_offering", offering.Name)
				assert.True(t, offering.Available)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceOfferingsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_offerings", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if names := query.Get("names"); names != "" {
			assert.Equal(t, "offering1,offering2", names)
		}

		if brokerGuids := query.Get("service_broker_guids"); brokerGuids != "" {
			assert.Equal(t, "broker-1,broker-2", brokerGuids)
		}

		if available := query.Get("available"); available != "" {
			assert.Equal(t, "true", available)
		}

		response := capi.ListResponse[capi.ServiceOffering]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/service_offerings?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/service_offerings?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.ServiceOffering{
				{
					Resource: capi.Resource{
						GUID:      "offering-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:        "database-service",
					Description: "A database service",
					Available:   true,
					Tags:        []string{"database", "sql"},
					Requires:    []string{},
					Shareable:   true,
					BrokerCatalog: capi.ServiceOfferingCatalog{
						ID: "catalog-id-1",
						Features: capi.ServiceOfferingCatalogFeatures{
							Bindable: true,
						},
					},
					Relationships: capi.ServiceOfferingRelationships{
						ServiceBroker: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "broker-1",
							},
						},
					},
				},
				{
					Resource: capi.Resource{
						GUID:      "offering-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:        "cache-service",
					Description: "A caching service",
					Available:   false,
					Tags:        []string{"cache", "memory"},
					Requires:    []string{"route_forwarding"},
					Shareable:   false,
					BrokerCatalog: capi.ServiceOfferingCatalog{
						ID: "catalog-id-2",
						Features: capi.ServiceOfferingCatalogFeatures{
							Bindable:             true,
							InstancesRetrievable: true,
						},
					},
					Relationships: capi.ServiceOfferingRelationships{
						ServiceBroker: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "broker-2",
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
	result, err := client.ServiceOfferings().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "offering-1", result.Resources[0].GUID)
	assert.Equal(t, "database-service", result.Resources[0].Name)
	assert.True(t, result.Resources[0].Available)
	assert.Equal(t, "offering-2", result.Resources[1].GUID)
	assert.Equal(t, "cache-service", result.Resources[1].Name)
	assert.False(t, result.Resources[1].Available)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"names":                {"offering1", "offering2"},
			"service_broker_guids": {"broker-1", "broker-2"},
			"available":            {"true"},
		},
	}
	result, err = client.ServiceOfferings().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceOfferingsClient_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		request      *capi.ServiceOfferingUpdateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "successful update",
			guid:         "test-offering-guid",
			expectedPath: "/v3/service_offerings/test-offering-guid",
			statusCode:   http.StatusOK,
			request: &capi.ServiceOfferingUpdateRequest{
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
					Annotations: map[string]string{
						"note": "Updated offering",
					},
				},
			},
			response: capi.ServiceOffering{
				Resource: capi.Resource{
					GUID:      "test-offering-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:        "my_service_offering",
				Description: "Provides my service",
				Available:   true,
				Tags:        []string{"relational", "caching"},
				Requires:    []string{},
				Shareable:   true,
				BrokerCatalog: capi.ServiceOfferingCatalog{
					ID: "catalog-id",
					Features: capi.ServiceOfferingCatalogFeatures{
						Bindable: true,
					},
				},
				Relationships: capi.ServiceOfferingRelationships{
					ServiceBroker: capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "broker-guid",
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
					Annotations: map[string]string{
						"note": "Updated offering",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "offering not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_offerings/non-existent-guid",
			statusCode:   http.StatusNotFound,
			request: &capi.ServiceOfferingUpdateRequest{
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
						"detail": "Service offering not found",
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

				var requestBody capi.ServiceOfferingUpdateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			offering, err := client.ServiceOfferings().Update(context.Background(), testCase.guid, testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, offering)
			} else {
				require.NoError(t, err)
				require.NotNil(t, offering)
				assert.Equal(t, testCase.guid, offering.GUID)
				assert.Equal(t, "my_service_offering", offering.Name)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceOfferingsClient_Delete(t *testing.T) {
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
			guid:         "test-offering-guid",
			expectedPath: "/v3/service_offerings/test-offering-guid",
			statusCode:   http.StatusNoContent,
			wantErr:      false,
		},
		{
			name:         "offering not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/service_offerings/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Service offering not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
		{
			name:         "offering has service instances",
			guid:         "offering-with-instances",
			expectedPath: "/v3/service_offerings/offering-with-instances",
			statusCode:   http.StatusUnprocessableEntity,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10008,
						"title":  "CF-UnprocessableEntity",
						"detail": "Service offering has service instances",
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

			err = client.ServiceOfferings().Delete(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServiceOfferingsClient_DeleteWithPurge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "DELETE", request.Method)
		assert.Equal(t, "true", request.URL.Query().Get("purge"))
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	offerings := NewServiceOfferingsClient(httpClient)

	err := offerings.Delete(context.Background(), "offering-guid", capi.PurgeServiceOffering)
	require.NoError(t, err)
}
