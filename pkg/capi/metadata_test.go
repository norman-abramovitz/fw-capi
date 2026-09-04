package capi_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

func TestMetadata_SetAndRemoveLabel(t *testing.T) {
	t.Parallel()

	meta := &capi.Metadata{}
	meta.SetLabel("example.com", "team", "platform")
	meta.SetLabel("", "env", "prod")

	require.NotNil(t, meta.Labels["example.com/team"])
	assert.Equal(t, "platform", *meta.Labels["example.com/team"])
	require.NotNil(t, meta.Labels["env"])
	assert.Equal(t, "prod", *meta.Labels["env"])

	meta.RemoveLabel("example.com", "team")
	meta.RemoveLabel("", "env")

	require.Contains(t, meta.Labels, "example.com/team")
	assert.Nil(t, meta.Labels["example.com/team"])
	require.Contains(t, meta.Labels, "env")
	assert.Nil(t, meta.Labels["env"])
}

func TestMetadata_SetAndRemoveAnnotation(t *testing.T) {
	t.Parallel()

	meta := &capi.Metadata{}
	meta.SetAnnotation("example.com", "contact", "team@example.com")
	meta.RemoveAnnotation("", "notes")

	require.NotNil(t, meta.Annotations["example.com/contact"])
	assert.Equal(t, "team@example.com", *meta.Annotations["example.com/contact"])
	require.Contains(t, meta.Annotations, "notes")
	assert.Nil(t, meta.Annotations["notes"])
}

func TestMetadata_RemovalMarshalsToNull(t *testing.T) {
	t.Parallel()

	meta := &capi.Metadata{}
	meta.SetLabel("", "keep", "value")
	meta.RemoveLabel("", "drop")
	meta.RemoveAnnotation("", "stale")

	data, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"labels":{"keep":"value","drop":null},"annotations":{"stale":null}}`,
		string(data))
}

func TestStringMap(t *testing.T) {
	t.Parallel()

	assert.Nil(t, capi.StringMap(nil))

	converted := capi.StringMap(map[string]string{"env": "prod", "team": "platform"})
	require.Len(t, converted, 2)
	require.NotNil(t, converted["env"])
	assert.Equal(t, "prod", *converted["env"])
	require.NotNil(t, converted["team"])
	assert.Equal(t, "platform", *converted["team"])
}
