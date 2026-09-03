//nolint:testpackage // RunE behavior tests need the unexported newClientFunc seam
package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// This file provides a small harness for exercising command RunE handlers
// end-to-end: it drives a real cobra command tree, substitutes a fake CAPI
// client through the newClientFunc seam, pins the output format, and captures
// what the handler writes to os.Stdout. These tests complement the
// structure-only tests (which assert on Use/Short/flag wiring) by proving that
// the handlers parse arguments, call the client correctly, format output, and
// propagate errors.
//
// Tests using this harness must NOT call t.Parallel(): captureStdout swaps the
// process-global os.Stdout and the helpers mutate package-global state
// (newClientFunc, viper). Each helper restores the prior value via t.Cleanup.

// fakeClient embeds capi.Client so it satisfies the full interface; individual
// tests override only the sub-client accessors they need. Any accessor left
// unset will panic if called, which surfaces unexpected client usage.
type fakeClient struct {
	capi.Client

	sidecars          capi.SidecarsClient
	isolationSegments capi.IsolationSegmentsClient
	droplets          capi.DropletsClient
}

func (f *fakeClient) Sidecars() capi.SidecarsClient {
	if f.sidecars == nil {
		panic("fakeClient.Sidecars() called but no stub was configured")
	}

	return f.sidecars
}

func (f *fakeClient) IsolationSegments() capi.IsolationSegmentsClient {
	if f.isolationSegments == nil {
		panic("fakeClient.IsolationSegments() called but no stub was configured")
	}

	return f.isolationSegments
}

func (f *fakeClient) Droplets() capi.DropletsClient {
	if f.droplets == nil {
		panic("fakeClient.Droplets() called but no stub was configured")
	}

	return f.droplets
}

// withStubClient installs client as the value returned by CreateClientWithAPI
// for the duration of the test.
func withStubClient(t *testing.T, client capi.Client) {
	t.Helper()

	original := newClientFunc
	newClientFunc = func(string) (capi.Client, error) { return client, nil }

	t.Cleanup(func() { newClientFunc = original })
}

// withClientError makes CreateClientWithAPI fail with err, exercising the
// client-construction error path that every handler shares.
func withClientError(t *testing.T, err error) {
	t.Helper()

	original := newClientFunc
	newClientFunc = func(string) (capi.Client, error) { return nil, err }

	t.Cleanup(func() { newClientFunc = original })
}

// withOutputFormat pins viper's "output" key (table/json/yaml) for the test,
// mirroring the --output persistent flag binding done in cmd/capi/main.go.
func withOutputFormat(t *testing.T, format string) {
	t.Helper()

	original := viper.GetString("output")

	viper.Set("output", format)
	t.Cleanup(func() { viper.Set("output", original) })
}

// newTestRootCommand wires sub beneath a root that registers the persistent
// flags RunE handlers read (api, output), mirroring cmd/capi/main.go so that
// cmd.Flag("api") resolves during execution.
func newTestRootCommand(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{
		Use:           "capi",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringP("api", "a", "https://api.example.test", "")
	root.PersistentFlags().String("output", "table", "")
	root.AddCommand(sub)

	return root
}

// captureStdout redirects os.Stdout for the duration of action and returns
// everything written to it. It is not safe for concurrent use.
func captureStdout(t *testing.T, action func()) string {
	t.Helper()

	original := os.Stdout

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer

	collected := make(chan string, 1)

	go func() {
		var buf bytes.Buffer

		_, _ = io.Copy(&buf, reader)
		collected <- buf.String()
	}()

	action()

	_ = writer.Close()

	os.Stdout = original

	return <-collected
}

// runCommand wires sub under a test root, sets argv, and executes it while
// capturing stdout. It returns the captured output and the RunE error.
func runCommand(t *testing.T, sub *cobra.Command, args ...string) (string, error) {
	t.Helper()

	root := newTestRootCommand(sub)
	root.SetArgs(append([]string{sub.Name()}, args...))

	var runErr error

	out := captureStdout(t, func() {
		runErr = root.Execute()
	})

	//nolint:wrapcheck // test harness intentionally returns the raw RunE error so callers can assert on sentinels
	return out, runErr
}

// recordingSidecarsClient is a capi.SidecarsClient stub that records the GUIDs
// it is called with and returns preconfigured results, so tests can assert both
// on output and on the exact client interaction.
type recordingSidecarsClient struct {
	capi.SidecarsClient

	getResult *capi.Sidecar
	getErr    error
	getGUID   string

	deleteErr  error
	deleteGUID string
}

func (s *recordingSidecarsClient) Get(_ context.Context, guid string) (*capi.Sidecar, error) {
	s.getGUID = guid

	return s.getResult, s.getErr
}

func (s *recordingSidecarsClient) Delete(_ context.Context, guid string) error {
	s.deleteGUID = guid

	return s.deleteErr
}

// recordingIsoSegmentsClient stubs capi.IsolationSegmentsClient for the
// Get-then-List-by-name lookup performed by findIsolationSegmentByNameOrGUID.
type recordingIsoSegmentsClient struct {
	capi.IsolationSegmentsClient

	getResult *capi.IsolationSegment
	getErr    error
	getGUID   string

	listResult *capi.ListResponse[capi.IsolationSegment]
	listErr    error
	listCalled bool
}

func (s *recordingIsoSegmentsClient) Get(_ context.Context, guid string) (*capi.IsolationSegment, error) {
	s.getGUID = guid

	return s.getResult, s.getErr
}

func (s *recordingIsoSegmentsClient) List(_ context.Context, _ *capi.QueryParams, _ ...capi.IsolationSegmentListOption) (*capi.ListResponse[capi.IsolationSegment], error) {
	s.listCalled = true

	return s.listResult, s.listErr
}

// recordingDropletsClient stubs capi.DropletsClient, recording the
// QueryParams passed to List so tests can assert on the filters a command
// builds (e.g. the --current flag on `capi apps droplets`).
type recordingDropletsClient struct {
	capi.DropletsClient

	listResult *capi.ListResponse[capi.Droplet]
	listErr    error
	listParams *capi.QueryParams
}

func (s *recordingDropletsClient) List(_ context.Context, params *capi.QueryParams, _ ...capi.DropletListOption) (*capi.ListResponse[capi.Droplet], error) {
	s.listParams = params

	return s.listResult, s.listErr
}
