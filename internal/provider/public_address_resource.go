package provider

import (
    "context"
    "github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*PublicAddressResource)(nil)
var _ resource.ResourceWithConfigure = (*PublicAddressResource)(nil)
var _ resource.ResourceWithImportState = (*PublicAddressResource)(nil)

type PublicAddressResource struct { client *polyfin.Client }
type publicAddressModel struct {
    ID types.String `tfsdk:"id"`
    PublicAddress types.String `tfsdk:"public_address"`
}

func NewPublicAddressResource() resource.Resource { return &PublicAddressResource{} }
func (r *PublicAddressResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_public_address"
}
func (r *PublicAddressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = rschema.Schema{
        Description: "Own only Polyfin's canonical Public address; OIDC identities, secrets and all other server settings remain unmanaged.",
        Attributes: map[string]rschema.Attribute{
            "id": rschema.StringAttribute{Computed: true, Description: "Fixed singleton identity: server."},
            "public_address": rschema.StringAttribute{Required: true, Description: "Canonical HTTPS origin, with no trailing slash or path."},
        },
    }
}
func (r *PublicAddressResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil { return }
    c, ok := req.ProviderData.(*polyfin.Client)
    if !ok { resp.Diagnostics.AddError("Invalid provider", "Polyfin client missing"); return }
    r.client = c
}
func (r *PublicAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var m publicAddressModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
    if resp.Diagnostics.HasError() { return }
    if r.client == nil { resp.Diagnostics.AddError("Missing client", "Configure Polyfin provider."); return }
    if err := r.client.SetPublicAddress(ctx, m.PublicAddress.ValueString()); err != nil {
        resp.Diagnostics.AddError("Public address update refused", err.Error()); return
    }
    m.ID = types.StringValue("server")
    resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *PublicAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var m publicAddressModel
    resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
    if resp.Diagnostics.HasError() { return }
    if r.client == nil { resp.Diagnostics.AddError("Missing client", "Configure Polyfin provider."); return }
    address, err := r.client.GetPublicAddress(ctx)
    if err != nil { resp.Diagnostics.AddError("Public address read failed", err.Error()); return }
    m.ID = types.StringValue("server")
    m.PublicAddress = types.StringValue(address)
    resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *PublicAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var m publicAddressModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
    if resp.Diagnostics.HasError() { return }
    if r.client == nil { resp.Diagnostics.AddError("Missing client", "Configure Polyfin provider."); return }
    if err := r.client.SetPublicAddress(ctx, m.PublicAddress.ValueString()); err != nil {
        resp.Diagnostics.AddError("Public address update refused", err.Error()); return
    }
    m.ID = types.StringValue("server")
    resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
func (r *PublicAddressResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
    resp.Diagnostics.AddError("Deletion blocked", "Public address is a singleton; it will not be cleared or reset during Terraform destroy. Remove the resource from state only after an explicit operator-approved handoff.")
}
func (r *PublicAddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    if req.ID != "server" {
        resp.Diagnostics.AddError("Invalid import ID", "Import this singleton with the exact ID: server")
        return
    }
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
