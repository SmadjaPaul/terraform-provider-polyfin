package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakePolyfin is a local stand-in for the Polyfin administration API. No real
// Polyfin is involved: every request is recorded so a test can prove that a
// code path (destroy, for instance) did or did not reach the server.
type fakePolyfin struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string
}

func newFakePolyfin(t *testing.T) *fakePolyfin {
	t.Helper()
	f := &fakePolyfin{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.mu.Unlock()
		if r.Header.Get("Cookie") != "polyfin_session=synthetic" {
			t.Errorf("missing synthetic admin cookie on %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/api/session":
			fmt.Fprint(w, `{"user":{"isAdministrator":true}}`)
		case "/admin/api/settings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"serverName":              "Polyfin",
				"quickConnectEnabled":     true,
				"legacyAuthorization":     false,
				"language":                "en",
				"publicAddress":           "https://tv.example.org",
				"defaultAudioLanguage":    "eng",
				"defaultSubtitleLanguage": "fre",
				"sendAppsToSource":        false,
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

// seen returns the requests the fake server received so far.
func (f *fakePolyfin) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

// forget drops the recorded requests, so a test can assert that a single
// code path reached the server (or did not).
func (f *fakePolyfin) forget() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = nil
}

// fakeCookieFile writes the synthetic administrator cookie the client needs.
// It lives in the test temporary directory and is never committed.
func fakeCookieFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cookie")
	if err := os.WriteFile(p, []byte("polyfin_session=synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

// singletonCase describes one server-wide singleton resource.
type singletonCase struct {
	typeName    string
	newResource func() resource.Resource
	emptyModel  func() any
}

func singletonCases() []singletonCase {
	return []singletonCase{
		{
			typeName:    "polyfin_public_address",
			newResource: NewPublicAddressResource,
			emptyModel: func() any {
				return &publicAddressModel{ID: types.StringNull(), PublicAddress: types.StringNull()}
			},
		},
		{
			typeName:    "polyfin_playback_languages",
			newResource: NewPlaybackLanguagesResource,
			emptyModel: func() any {
				return &playbackLanguagesModel{ID: types.StringNull(), Audio: types.StringNull(), Subtitles: types.StringNull()}
			},
		},
		{
			typeName:    "polyfin_playback_source_policy",
			newResource: NewPlaybackSourcePolicyResource,
			emptyModel: func() any {
				return &playbackSourcePolicyModel{ID: types.StringNull(), SendAppsToSource: types.BoolNull()}
			},
		},
	}
}

// configureSingleton gives the resource a client wired to the fake server and
// checks that the provider data was accepted.
func configureSingleton(t *testing.T, r resource.Resource, fake *fakePolyfin) {
	t.Helper()
	client, err := polyfin.New(fake.URL, fakeCookieFile(t))
	if err != nil {
		t.Fatal(err)
	}
	configurable, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("the resource does not accept a provider client")
	}
	var resp resource.ConfigureResponse
	configurable.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

// singletonState builds the state the framework hands to a resource before a
// read or an import, so ImportStatePassthroughID has a schema to write to.
func singletonState(t *testing.T, r resource.Resource, model any) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

// requireErrorDiagnostic checks that the code path refused the operation with
// the expected diagnostic.
func requireErrorDiagnostic(t *testing.T, diags diag.Diagnostics, summary string) {
	t.Helper()
	for _, d := range diags.Errors() {
		if d.Summary() == summary {
			return
		}
	}
	t.Fatalf("missing error diagnostic %q: %v", summary, diags)
}

// The three server-wide singletons are never destroyed: a destroy is a hard
// error and must not reach the Polyfin API at all.
func TestSingletonResourcesRefuseDestroyWithoutCallingTheServer(t *testing.T) {
	for _, tc := range singletonCases() {
		t.Run(tc.typeName, func(t *testing.T) {
			ctx := context.Background()
			fake := newFakePolyfin(t)
			r := tc.newResource()
			configureSingleton(t, r, fake)

			// Liveness: the resource really reads the fake server, so a
			// silent destroy cannot be mistaken for a broken client.
			state := singletonState(t, r, tc.emptyModel())
			var read resource.ReadResponse
			read.State = state
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if read.Diagnostics.HasError() {
				t.Fatalf("read failed: %v", read.Diagnostics)
			}
			if len(fake.seen()) == 0 {
				t.Fatal("the resource never reached the fake server")
			}
			fake.forget()

			var del resource.DeleteResponse
			r.Delete(ctx, resource.DeleteRequest{}, &del)
			requireErrorDiagnostic(t, del.Diagnostics, "Deletion blocked")
			if got := fake.seen(); len(got) != 0 {
				t.Fatalf("destroy called the Polyfin API: %v", got)
			}
		})
	}
}

// A singleton is imported with its fixed identity "server" only; every other
// identifier is refused and leaves no state behind.
func TestSingletonResourcesImportOnlyTheServerIdentifier(t *testing.T) {
	for _, tc := range singletonCases() {
		t.Run(tc.typeName, func(t *testing.T) {
			ctx := context.Background()
			r := tc.newResource()
			importer, ok := r.(resource.ResourceWithImportState)
			if !ok {
				t.Fatal("the resource does not support import")
			}

			accepted := resource.ImportStateResponse{State: singletonState(t, r, tc.emptyModel())}
			importer.ImportState(ctx, resource.ImportStateRequest{ID: "server"}, &accepted)
			if accepted.Diagnostics.HasError() {
				t.Fatalf("import of the singleton identity failed: %v", accepted.Diagnostics)
			}
			var id types.String
			if diags := accepted.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if id.ValueString() != "server" {
				t.Fatalf("imported identity %q, want %q", id.ValueString(), "server")
			}

			for _, refused := range []string{"", "Server", "SERVER", " server", "server ", "servers", "server/settings", "0", "id", "polyfin_public_address"} {
				resp := resource.ImportStateResponse{State: singletonState(t, r, tc.emptyModel())}
				importer.ImportState(ctx, resource.ImportStateRequest{ID: refused}, &resp)
				requireErrorDiagnostic(t, resp.Diagnostics, "Invalid import ID")
				var got types.String
				if diags := resp.State.GetAttribute(ctx, path.Root("id"), &got); diags.HasError() {
					t.Fatal(diags)
				}
				if !got.IsNull() {
					t.Fatalf("refused import %q still wrote the identity %q", refused, got.ValueString())
				}
			}
		})
	}
}

// The provider exposes the three singletons under stable type names.
func TestSingletonResourceTypeNames(t *testing.T) {
	for _, tc := range singletonCases() {
		var resp resource.MetadataResponse
		tc.newResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "polyfin"}, &resp)
		if resp.TypeName != tc.typeName {
			t.Fatalf("type name %q, want %q", resp.TypeName, tc.typeName)
		}
	}
}
