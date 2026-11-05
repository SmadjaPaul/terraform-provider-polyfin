package provider

import (
	"context"
	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"net/url"
	"strings"
)

var _ resource.Resource = (*AddonResource)(nil)
var _ resource.ResourceWithConfigure = (*AddonResource)(nil)
var _ resource.ResourceWithImportState = (*AddonResource)(nil)

type AddonResource struct{ client *polyfin.Client }
type addonModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Kind            types.String `tfsdk:"kind"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	ManifestURL     types.String `tfsdk:"manifest_url_wo"`
	ManifestVersion types.Int64  `tfsdk:"manifest_url_wo_version"`
}

func NewAddonResource() resource.Resource { return &AddonResource{} }
func (r *AddonResource) Metadata(_ context.Context, q resource.MetadataRequest, a *resource.MetadataResponse) {
	a.TypeName = q.ProviderTypeName + "_addon"
}
func (r *AddonResource) Schema(_ context.Context, _ resource.SchemaRequest, a *resource.SchemaResponse) {
	a.Schema = rschema.Schema{Description: "Native Polyfin server addon. Removal is blocked to prevent data loss.", Attributes: map[string]rschema.Attribute{
		"id":                      rschema.StringAttribute{Computed: true, Description: "Polyfin-local installation UUID."},
		"name":                    rschema.StringAttribute{Required: true, Description: "Exact manifest name; immutable after creation."},
		"kind":                    rschema.StringAttribute{Computed: true, Description: "Native stremio or eclipse source kind."},
		"enabled":                 rschema.BoolAttribute{Required: true, Description: "Whether this addon is enabled."},
		"manifest_url_wo":         rschema.StringAttribute{Optional: true, WriteOnly: true, Sensitive: true, Description: "Tokenized manifest address; write-only, never stored in state; mandatory on create and on deliberate version increment."},
		"manifest_url_wo_version": rschema.Int64Attribute{Optional: true, Description: "Non-sensitive rotation counter. Increment when changing manifest_url_wo."},
	}}
}
func (r *AddonResource) Configure(_ context.Context, q resource.ConfigureRequest, a *resource.ConfigureResponse) {
	if q.ProviderData == nil {
		return
	}
	c, ok := q.ProviderData.(*polyfin.Client)
	if !ok {
		a.Diagnostics.AddError("Invalid provider", "Client missing")
		return
	}
	r.client = c
}
func (r *AddonResource) auth(ctx context.Context) error { return r.client.CheckAdmin(ctx) }

// Allow plain HTTP only for exact Kubernetes Service FQDNs.
// Polyfin restricts server-side private addon installation to administrators.
func isKubernetesServiceHost(host string) bool {
	labels := strings.Split(strings.ToLower(host), ".")
	if len(labels) != 5 || labels[2] != "svc" || labels[3] != "cluster" || labels[4] != "local" {
		return false
	}
	for _, label := range labels[:2] {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
				return false
			}
		}
	}
	return true
}

func validateURL(s string) bool {
	parsed, err := url.Parse(s)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" ||
		!strings.HasSuffix(parsed.Path, "/manifest.json") ||
		strings.IndexByte(s, 0) >= 0 || strings.IndexByte(s, 10) >= 0 || strings.IndexByte(s, 13) >= 0 {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return parsed.Scheme == "http" && parsed.Port() == "" && isKubernetesServiceHost(parsed.Hostname())
}

func (r *AddonResource) Create(ctx context.Context, q resource.CreateRequest, a *resource.CreateResponse) {
	var m addonModel
	var config addonModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	a.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider.")
		return
	}
	if err := r.auth(ctx); err != nil {
		a.Diagnostics.AddError("Not authorized", err.Error())
		return
	}
	secret := config.ManifestURL.ValueString()
	if config.ManifestURL.IsNull() || config.ManifestURL.IsUnknown() || !validateURL(secret) {
		a.Diagnostics.AddError("Manifest required", "Supply an HTTPS manifest or in-cluster Kubernetes Service HTTP manifest_url_wo; value suppressed.")
		return
	}
	existing, err := r.client.List(ctx)
	if err != nil {
		a.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	for _, v := range existing {
		if strings.EqualFold(v.Name, m.Name.ValueString()) {
			a.Diagnostics.AddError("Addon already exists", "Refusing duplicate installation; import the existing Polyfin UUID instead.")
			return
		}
	}
	created, err := r.client.Install(ctx, secret)
	if err != nil {
		a.Diagnostics.AddError("Install failed", err.Error())
		return
	}
	if created.Name != m.Name.ValueString() {
		a.Diagnostics.AddError("Manifest identity mismatch", "Installed addon name differs from plan; inspect native admin UI before retrying.")
		return
	}
	if !m.Enabled.ValueBool() {
		created, err = r.client.SetEnabled(ctx, created.ID, false)
		if err != nil {
			a.Diagnostics.AddError("Disable failed", err.Error())
			return
		}
	}
	m.ID = types.StringValue(created.ID)
	m.Kind = types.StringValue(created.Kind)
	m.ManifestURL = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}
func (r *AddonResource) Read(ctx context.Context, q resource.ReadRequest, a *resource.ReadResponse) {
	var m addonModel
	a.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider.")
		return
	}
	if err := r.auth(ctx); err != nil {
		a.Diagnostics.AddError("Not authorized", err.Error())
		return
	}
	found, ok, err := r.client.Find(ctx, m.ID.ValueString())
	if err != nil {
		a.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	if !ok {
		a.Diagnostics.AddError("Addon missing", "Managed addon was removed remotely; refusing implicit recreation or state removal.")
		return
	}
	m.Name = types.StringValue(found.Name)
	m.Kind = types.StringValue(found.Kind)
	m.Enabled = types.BoolValue(found.Enabled)
	m.ManifestURL = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}
func (r *AddonResource) Update(ctx context.Context, q resource.UpdateRequest, a *resource.UpdateResponse) {
	var plan, state, config addonModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &plan)...)
	a.Diagnostics.Append(q.State.Get(ctx, &state)...)
	if a.Diagnostics.HasError() {
		return
	}
	a.Diagnostics.Append(q.Config.Get(ctx, &config)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider.")
		return
	}
	if plan.Name.ValueString() != state.Name.ValueString() {
		a.Diagnostics.AddError("Rename not supported", "Addon names must not change in place.")
		return
	}
	if err := r.auth(ctx); err != nil {
		a.Diagnostics.AddError("Not authorized", err.Error())
		return
	}
	found, ok, err := r.client.Find(ctx, state.ID.ValueString())
	if err != nil {
		a.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	if !ok {
		a.Diagnostics.AddError("Addon missing", "Remote addon no longer exists.")
		return
	}
	if !plan.ManifestVersion.Equal(state.ManifestVersion) {
		url := config.ManifestURL.ValueString()
		if config.ManifestURL.IsNull() || config.ManifestURL.IsUnknown() || !validateURL(url) {
			a.Diagnostics.AddError("Rotation secret missing", "Incrementing version requires manifest_url_wo; value suppressed.")
			return
		}
		found, err = r.client.Replace(ctx, state.ID.ValueString(), url)
		if err != nil {
			a.Diagnostics.AddError("Replace failed", err.Error())
			return
		}
		if found.ID != state.ID.ValueString() {
			a.Diagnostics.AddError("Addon replaced unexpectedly", "Native UUID changed; stop and inspect.")
			return
		}
	}
	if found.Enabled != plan.Enabled.ValueBool() {
		found, err = r.client.SetEnabled(ctx, state.ID.ValueString(), plan.Enabled.ValueBool())
		if err != nil {
			a.Diagnostics.AddError("Update failed", err.Error())
			return
		}
	}
	plan.ID = state.ID
	plan.Kind = types.StringValue(found.Kind)
	plan.ManifestURL = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &plan)...)
}
func (r *AddonResource) Delete(ctx context.Context, _ resource.DeleteRequest, a *resource.DeleteResponse) {
	a.Diagnostics.AddError("Deletion blocked", "Removing a polyfin_addon resource never deletes its server addon. Keep resource in config or use an explicit, reviewed manual decommission process.")
}
func (r *AddonResource) ImportState(ctx context.Context, q resource.ImportStateRequest, a *resource.ImportStateResponse) {
	a.Diagnostics.Append(a.State.SetAttribute(ctx, path.Root("id"), q.ID)...)
}
