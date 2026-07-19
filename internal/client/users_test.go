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

// testUsersUserPath is the path for the "user-guid" fixture user.
const testUsersUserPath = "/v3/users/user-guid"

func TestUsersClient_Create_WithGUID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/users", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var req capi.UserCreateRequest

		err := json.NewDecoder(request.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, testUserGUID, req.GUID)

		user := capi.User{
			Resource: capi.Resource{
				GUID:      testUserGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testUsersUserPath,
					},
				},
			},
			Username:         testUserNameFixture,
			PresentationName: testUserNameFixture,
			Origin:           testUAAOrigin,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(user)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	req := &capi.UserCreateRequest{
		GUID: testUserGUID,
	}

	user, err := users.Create(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUserGUID, user.GUID)
	assert.Equal(t, testUserNameFixture, user.Username)
	assert.Equal(t, testUAAOrigin, user.Origin)
}

func TestUsersClient_Create_WithUsernameAndOrigin(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/users", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var req capi.UserCreateRequest

		err := json.NewDecoder(request.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, testUserNameFixture, req.Username)
		assert.Equal(t, "ldap", req.Origin)

		user := capi.User{
			Resource: capi.Resource{
				GUID:      "generated-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: "/v3/users/generated-guid",
					},
				},
			},
			Username:         testUserNameFixture,
			PresentationName: testUserNameFixture,
			Origin:           "ldap",
			Metadata: &capi.Metadata{
				Labels:      capi.StringMap(map[string]string{}),
				Annotations: capi.StringMap(map[string]string{}),
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(user)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	req := &capi.UserCreateRequest{
		Username: testUserNameFixture,
		Origin:   "ldap",
		Metadata: &capi.Metadata{
			Labels:      capi.StringMap(map[string]string{}),
			Annotations: capi.StringMap(map[string]string{}),
		},
	}

	user, err := users.Create(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "generated-guid", user.GUID)
	assert.Equal(t, testUserNameFixture, user.Username)
	assert.Equal(t, "ldap", user.Origin)
}

func TestUsersClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testUsersUserPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		user := capi.User{
			Resource: capi.Resource{
				GUID:      testUserGUID,
				CreatedAt: time.Now().Add(-time.Hour),
				UpdatedAt: time.Now().Add(-30 * time.Minute),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testUsersUserPath,
					},
				},
			},
			Username:         testUserNameFixture,
			PresentationName: testUserNameFixture,
			Origin:           testUAAOrigin,
			Metadata: &capi.Metadata{
				Labels: capi.StringMap(map[string]string{
					testEnvironmentLabelKey: testProductionLabel,
				}),
				Annotations: capi.StringMap(map[string]string{
					testNoteAnnotationKey: "admin user",
				}),
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(user)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	user, err := users.Get(context.Background(), testUserGUID)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUserGUID, user.GUID)
	assert.Equal(t, testUserNameFixture, user.Username)
	assert.Equal(t, testUAAOrigin, user.Origin)
	assert.Equal(t, testProductionLabel, capi.StringValue(user.Metadata.Labels[testEnvironmentLabelKey]))
}

func TestUsersClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/users", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, testUserNameFixture, request.URL.Query().Get("usernames"))
		assert.Equal(t, "2", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.User]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First: capi.Link{
					Href: "/v3/users?page=1&per_page=2",
				},
				Last: capi.Link{
					Href: "/v3/users?page=1&per_page=2",
				},
			},
			Resources: []capi.User{
				{
					Resource: capi.Resource{
						GUID:      "user-guid-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Links: capi.Links{
							testSelfKey: capi.Link{
								Href: "/v3/users/user-guid-1",
							},
						},
					},
					Username:         testUserNameFixture,
					PresentationName: testUserNameFixture,
					Origin:           testUAAOrigin,
				},
				{
					Resource: capi.Resource{
						GUID:      testClientIDFixture,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Links: capi.Links{
							testSelfKey: capi.Link{
								Href: "/v3/users/client-id",
							},
						},
					},
					Username:         "",
					PresentationName: testClientIDFixture,
					Origin:           "",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	params := &capi.QueryParams{
		PerPage: 2,
		Filters: map[string][]string{
			"usernames": {testUserNameFixture},
		},
	}

	list, err := users.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, testUserNameFixture, list.Resources[0].Username)
	assert.Empty(t, list.Resources[1].Username) // UAA client
	assert.Equal(t, testClientIDFixture, list.Resources[1].PresentationName)
}

func TestUsersClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testUsersUserPath, request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var req capi.UserUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, testStagingLabel, capi.StringValue(req.Metadata.Labels[testEnvironmentLabelKey]))
		assert.Equal(t, "updated note", capi.StringValue(req.Metadata.Annotations[testNoteAnnotationKey]))

		user := capi.User{
			Resource: capi.Resource{
				GUID:      testUserGUID,
				CreatedAt: time.Now().Add(-time.Hour),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testUsersUserPath,
					},
				},
			},
			Username:         testUserNameFixture,
			PresentationName: testUserNameFixture,
			Origin:           testUAAOrigin,
			Metadata: &capi.Metadata{
				Labels: capi.StringMap(map[string]string{
					testEnvironmentLabelKey: testStagingLabel,
				}),
				Annotations: capi.StringMap(map[string]string{
					testNoteAnnotationKey: "updated note",
				}),
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(user)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	req := &capi.UserUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testEnvironmentLabelKey: testStagingLabel,
			}),
			Annotations: capi.StringMap(map[string]string{
				testNoteAnnotationKey: "updated note",
			}),
		},
	}

	user, err := users.Update(context.Background(), testUserGUID, req)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUserGUID, user.GUID)
	assert.Equal(t, testStagingLabel, capi.StringValue(user.Metadata.Labels[testEnvironmentLabelKey]))
	assert.Equal(t, "updated note", capi.StringValue(user.Metadata.Annotations[testNoteAnnotationKey]))
}

func TestUsersClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/users/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testUsersUserPath, request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "https://api.example.org/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	users := NewUsersClient(httpClient)

	job, err := users.Delete(context.Background(), testUserGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}
