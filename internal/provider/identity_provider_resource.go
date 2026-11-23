package provider

import (
	"context"

	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*IdentityProviderResource)(nil)
var _ resource.ResourceWithConfigure = (*IdentityProviderResource)(nil)
var _ resource.ResourceWithImportState = (*IdentityProviderResource)(nil)

// IdentityProviderResource owns only the OpenID Connect provider block of the
// Polyfin settings (Settings > Security > Identity provider). The client secret
// is write-only: it is sent when client_secret_wo_version changes and never
// enters the state or the plan output.
type IdentityProviderResource struct{ client *polyfin.Client }

type identityProviderModel struct {
	ID                  types.String `tfsdk:"id"`
	Issuer              types.String `tfsdk:"issuer"`
	ButtonName          types.String `tfsdk:"button_name"`
	ClientID            types.String `tfsdk:"client_id"`
	ClientSecretWO      types.String `tfsdk:"client_secret_wo"`
	ClientSecretVersion types.Int64  `tfsdk:"client_secret_wo_version"`
	ClientSecretSet     types.Bool   `tfsdk:"client_secret_set"`
	UserClaim           types.String `tfsdk:"user_claim"`
	CreateAccounts      types.Bool   `tfsdk:"create_accounts"`
	AdminClaim          types.String `tfsdk:"admin_claim"`
	AdminValue          types.String `tfsdk:"admin_value"`
	RedirectURI         types.String `tfsdk:"redirect_uri"`
}

func NewIdentityProviderResource() resource.Resource { return &IdentityProviderResource{} }

func (r *IdentityProviderResource) Metadata(_ context.Context, q resource.MetadataRequest, a *resource.MetadataResponse) {
	a.TypeName = q.ProviderTypeName + "_identity_provider"
}

func (r *IdentityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, a *resource.SchemaResponse) {
	a.Schema = schema.Schema{
		Description: "Polyfin's OpenID Connect identity provider (Settings > Security). Does not manage the public address, accounts or account links. Destroy is blocked to protect live sign-in.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "Singleton ID: server", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"issuer":      schema.StringAttribute{Required: true, Description: "Issuer URL exactly as the discovery document gives it, https, no query or fragment."},
			"button_name": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Text after \"Sign in with\" (64 bytes max); empty shows the issuer host."},
			"client_id":   schema.StringAttribute{Required: true, Description: "OIDC client ID."},
			"client_secret_wo": schema.StringAttribute{
				Optional: true, WriteOnly: true, Sensitive: true,
				Description: "Confidential client secret, sent only when client_secret_wo_version is set or changes; never stored in the state (needs OpenTofu 1.11+ or Terraform 1.11+).",
			},
			"client_secret_wo_version": schema.Int64Attribute{Optional: true, Description: "Increase to send client_secret_wo again (rotation)."},
			"client_secret_set":        schema.BoolAttribute{Computed: true, Description: "Whether Polyfin holds a client secret (read from the server)."},
			"user_claim":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("preferred_username"), Description: "ID token claim that names the account on first sign-in."},
			"create_accounts":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Create an account for someone no account is linked to."},
			"admin_claim":              schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Claim granting administrator rights; set together with admin_value."},
			"admin_value":              schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Value (or list member) of admin_claim that makes someone an administrator."},
			"redirect_uri":             schema.StringAttribute{Computed: true, Description: "Redirect URI to register at the provider (<public address>/admin/api/provider/callback)."},
		},
	}
}

func (r *IdentityProviderResource) Configure(_ context.Context, q resource.ConfigureRequest, a *resource.ConfigureResponse) {
	if q.ProviderData == nil {
		return
	}
	c, ok := q.ProviderData.(*polyfin.Client)
	if !ok {
		a.Diagnostics.AddError("Invalid provider", "Polyfin client missing")
		return
	}
	r.client = c
}

func (m identityProviderModel) toClient() polyfin.IdentityProvider {
	return polyfin.IdentityProvider{
		Issuer: m.Issuer.ValueString(), Name: m.ButtonName.ValueString(), ClientID: m.ClientID.ValueString(),
		UserClaim: m.UserClaim.ValueString(), CreateAccounts: m.CreateAccounts.ValueBool(),
		AdminClaim: m.AdminClaim.ValueString(), AdminValue: m.AdminValue.ValueString(),
	}
}

func (m *identityProviderModel) fromClient(p polyfin.IdentityProvider) {
	m.ID = types.StringValue("server")
	m.Issuer = types.StringValue(p.Issuer)
	m.ButtonName = types.StringValue(p.Name)
	m.ClientID = types.StringValue(p.ClientID)
	m.UserClaim = types.StringValue(p.UserClaim)
	m.CreateAccounts = types.BoolValue(p.CreateAccounts)
	m.AdminClaim = types.StringValue(p.AdminClaim)
	m.AdminValue = types.StringValue(p.AdminValue)
	m.ClientSecretSet = types.BoolValue(p.SecretSet)
	m.RedirectURI = types.StringValue(p.RedirectURI)
	m.ClientSecretWO = types.StringNull()
}

// write applies the plan; the secret travels only when sendSecret is true.
func (r *IdentityProviderResource) write(ctx context.Context, plan tfsdk.Plan, cfg tfsdk.Config, sendSecret bool, diags interface {
	AddError(string, string)
}) (identityProviderModel, bool) {
	var m identityProviderModel
	if d := plan.Get(ctx, &m); d.HasError() {
		diags.AddError("Invalid plan", "identity provider plan unreadable")
		return m, false
	}
	var secret types.String
	if d := cfg.GetAttribute(ctx, path.Root("client_secret_wo"), &secret); d.HasError() {
		diags.AddError("Invalid configuration", "client_secret_wo unreadable")
		return m, false
	}
	var sp *string
	if sendSecret && !secret.IsNull() {
		v := secret.ValueString()
		sp = &v
	}
	if r.client == nil {
		diags.AddError("Missing client", "Configure provider")
		return m, false
	}
	if err := r.client.SetIdentityProvider(ctx, m.toClient(), sp); err != nil {
		diags.AddError("Identity provider refused", err.Error())
		return m, false
	}
	got, err := r.client.GetIdentityProvider(ctx)
	if err != nil {
		diags.AddError("Identity provider read failed", err.Error())
		return m, false
	}
	version := m.ClientSecretVersion
	m.fromClient(got)
	m.ClientSecretVersion = version
	return m, true
}

func (r *IdentityProviderResource) Create(ctx context.Context, q resource.CreateRequest, a *resource.CreateResponse) {
	m, ok := r.write(ctx, q.Plan, q.Config, true, &a.Diagnostics)
	if ok {
		a.Diagnostics.Append(a.State.Set(ctx, &m)...)
	}
}

func (r *IdentityProviderResource) Update(ctx context.Context, q resource.UpdateRequest, a *resource.UpdateResponse) {
	var prior, planned identityProviderModel
	a.Diagnostics.Append(q.State.Get(ctx, &prior)...)
	a.Diagnostics.Append(q.Plan.Get(ctx, &planned)...)
	if a.Diagnostics.HasError() {
		return
	}
	send := !planned.ClientSecretVersion.Equal(prior.ClientSecretVersion)
	m, ok := r.write(ctx, q.Plan, q.Config, send, &a.Diagnostics)
	if ok {
		a.Diagnostics.Append(a.State.Set(ctx, &m)...)
	}
}

func (r *IdentityProviderResource) Read(ctx context.Context, q resource.ReadRequest, a *resource.ReadResponse) {
	var m identityProviderModel
	a.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	p, err := r.client.GetIdentityProvider(ctx)
	if err != nil {
		a.Diagnostics.AddError("Identity provider read failed", err.Error())
		return
	}
	version := m.ClientSecretVersion
	m.fromClient(p)
	m.ClientSecretVersion = version
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *IdentityProviderResource) Delete(_ context.Context, _ resource.DeleteRequest, a *resource.DeleteResponse) {
	a.Diagnostics.AddError("Identity provider destroy blocked", "Destroying would switch single sign-on off. Remove it from the state with a removed block, or change the settings in place.")
}

func (r *IdentityProviderResource) ImportState(ctx context.Context, q resource.ImportStateRequest, a *resource.ImportStateResponse) {
	if q.ID != "server" {
		a.Diagnostics.AddError("Invalid import ID", "The singleton ID is \"server\".")
		return
	}
	a.Diagnostics.Append(a.State.SetAttribute(ctx, path.Root("id"), "server")...)
}
