package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestIdentityProviderSchemaIsValid(t *testing.T) {
	r := NewIdentityProviderResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if d := resp.Schema.ValidateImplementation(context.Background()); d.HasError() {
		t.Fatal(d)
	}
	if !resp.Schema.Attributes["client_secret_wo"].IsWriteOnly() {
		t.Fatal("client secret must be write-only")
	}
}
