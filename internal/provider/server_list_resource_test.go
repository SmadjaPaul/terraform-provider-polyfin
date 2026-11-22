package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestServerListResourcesSchema(t *testing.T) {
	for name, tc := range map[string]struct {
		r         resource.Resource
		attribute string
	}{
		"polyfin_known_proxies": {NewKnownProxiesResource(), "entries"},
		"polyfin_version_order": {NewVersionOrderResource(), "criteria"},
	} {
		var meta resource.MetadataResponse
		tc.r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "polyfin"}, &meta)
		if meta.TypeName != name {
			t.Fatalf("type name %q, want %q", meta.TypeName, name)
		}
		var s resource.SchemaResponse
		tc.r.Schema(context.Background(), resource.SchemaRequest{}, &s)
		if s.Diagnostics.HasError() {
			t.Fatal(s.Diagnostics)
		}
		if _, ok := s.Schema.Attributes[tc.attribute]; !ok {
			t.Fatalf("%s misses %s", name, tc.attribute)
		}
		if _, ok := s.Schema.Attributes["id"]; !ok {
			t.Fatalf("%s misses id", name)
		}
		var del resource.DeleteResponse
		tc.r.Delete(context.Background(), resource.DeleteRequest{}, &del)
		if !del.Diagnostics.HasError() {
			t.Fatalf("%s destroy must be refused", name)
		}
	}
}

func TestServerListApplyRequiresCanonicalValue(t *testing.T) {
	r := NewKnownProxiesResource().(*ServerListResource)
	if err := r.apply(context.Background(), []string{"10.244.0.161/16"}); err == nil {
		t.Fatal("non-canonical known proxies accepted")
	}
	v := NewVersionOrderResource().(*ServerListResource)
	if err := v.apply(context.Background(), []string{"resolution-high", "resolution-low"}); err == nil {
		t.Fatal("invalid version order accepted")
	}
}
