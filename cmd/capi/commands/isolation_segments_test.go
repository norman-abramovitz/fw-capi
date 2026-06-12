//nolint:testpackage // needs access to unexported listSpacesForSegment
package commands

import (
	"context"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubIsolationSegmentsClient implements capi.IsolationSegmentsClient.
// Only ListSpaces is wired; other methods are not expected to be called.
type stubIsolationSegmentsClient struct {
	spaces *capi.ListResponse[capi.Space]
	err    error
}

func (s *stubIsolationSegmentsClient) Create(_ context.Context, _ *capi.IsolationSegmentCreateRequest) (*capi.IsolationSegment, error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) Get(_ context.Context, _ string) (*capi.IsolationSegment, error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) List(_ context.Context, _ *capi.QueryParams) (*capi.ListResponse[capi.IsolationSegment], error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) Update(_ context.Context, _ string, _ *capi.IsolationSegmentUpdateRequest) (*capi.IsolationSegment, error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) Delete(_ context.Context, _ string) error {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) EntitleOrganizations(_ context.Context, _ string, _ []string) (*capi.ToManyRelationship, error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) RevokeOrganization(_ context.Context, _ string, _ string) error {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) ListOrganizations(_ context.Context, _ string, _ *capi.QueryParams) (*capi.ListResponse[capi.Organization], error) {
	panic("not implemented")
}

func (s *stubIsolationSegmentsClient) ListSpaces(_ context.Context, _ string, _ *capi.QueryParams) (*capi.ListResponse[capi.Space], error) {
	return s.spaces, s.err
}

// stubClient satisfies capi.Client with only IsolationSegments() wired.
// All other methods panic — they must not be called by listSpacesForSegment.
type stubClient struct {
	capi.Client // embed to satisfy the interface; concrete methods below override
	isoSegments capi.IsolationSegmentsClient
}

func (s *stubClient) IsolationSegments() capi.IsolationSegmentsClient {
	return s.isoSegments
}

func TestListSpacesForSegment_ReturnsSpaces(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)

	want := &capi.ListResponse[capi.Space]{
		Resources: []capi.Space{
			{
				Resource: capi.Resource{
					GUID:      "space-guid-1",
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name: "my-space",
			},
		},
	}

	isoClient := &stubIsolationSegmentsClient{spaces: want}
	client := &stubClient{isoSegments: isoClient}

	got, err := listSpacesForSegment(client, "segment-guid-abc")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListSpacesForSegment_PropagatesError(t *testing.T) {
	t.Parallel()

	isoClient := &stubIsolationSegmentsClient{err: context.DeadlineExceeded}
	client := &stubClient{isoSegments: isoClient}

	got, err := listSpacesForSegment(client, "segment-guid-abc")

	require.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNewIsolationSegmentsCommand_HasListSpacesSubcommand(t *testing.T) {
	t.Parallel()

	cmd := NewIsolationSegmentsCommand()
	assert.Equal(t, "isolation-segments", cmd.Use)

	var found bool

	for _, sub := range cmd.Commands() {
		if sub.Name() == "list-spaces" {
			found = true

			break
		}
	}

	assert.True(t, found, "list-spaces subcommand must be present")
}
