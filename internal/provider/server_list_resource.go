package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// serverListSetting describes one list-valued server-wide Polyfin setting
// owned as a singleton resource (import ID "server", destroy refused).
type serverListSetting struct {
	typeSuffix  string
	attribute   string
	description string
	attrDesc    string
	// canonical returns the value as Polyfin stores it, or an error when the
	// configuration is invalid. Configurations must already be canonical so
	// that a refresh never produces a perpetual diff.
	canonical func([]string) ([]string, error)
	get       func(*polyfin.Client, context.Context) ([]string, error)
	set       func(*polyfin.Client, context.Context, []string) error
}

var _ resource.Resource = (*ServerListResource)(nil)
var _ resource.ResourceWithConfigure = (*ServerListResource)(nil)
var _ resource.ResourceWithImportState = (*ServerListResource)(nil)

type ServerListResource struct {
	client  *polyfin.Client
	setting serverListSetting
}

func NewKnownProxiesResource() resource.Resource {
	return &ServerListResource{setting: serverListSetting{
		typeSuffix:  "_known_proxies",
		attribute:   "entries",
		description: "Only Polyfin's server-wide Settings > Security > Known proxies: the reverse proxies whose forwarded headers name a request's client. Does not change any other setting.",
		attrDesc:    "IP addresses, CIDR networks (host bits cleared) then host names in lower case, in Polyfin's canonical order, 64 at most. An empty list trusts no forwarded header.",
		canonical:   polyfin.CanonicalKnownProxies,
		get:         (*polyfin.Client).GetKnownProxies,
		set:         (*polyfin.Client).SetKnownProxies,
	}}
}

func NewVersionOrderResource() resource.Resource {
	return &ServerListResource{setting: serverListSetting{
		typeSuffix:  "_version_order",
		attribute:   "criteria",
		description: "Only Polyfin's server-wide version order (Settings > Playback): the criteria ranking a title's versions. Codec, source and edition sub-orders are not changed.",
		attrDesc:    "Up to six criteria, one per kind, among language, resolution-high, resolution-low, size-large, size-small, size-per-minute, codec, hdr-first, hdr-last, source, tier, audio, edition, labels, cached, seeders. \"language\" ranks the user's preferred audio language first. Empty keeps the addons' order.",
		canonical: func(v []string) ([]string, error) {
			if err := polyfin.ValidateVersionOrder(v); err != nil {
				return nil, err
			}
			return v, nil
		},
		get: (*polyfin.Client).GetVersionOrder,
		set: (*polyfin.Client).SetVersionOrder,
	}}
}

func (r *ServerListResource) Metadata(_ context.Context, q resource.MetadataRequest, a *resource.MetadataResponse) {
	a.TypeName = q.ProviderTypeName + r.setting.typeSuffix
}

func (r *ServerListResource) Schema(_ context.Context, _ resource.SchemaRequest, a *resource.SchemaResponse) {
	a.Schema = schema.Schema{
		Description: r.setting.description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Singleton ID: server"},
			r.setting.attribute: schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: r.setting.attrDesc,
			},
		},
	}
}

func (r *ServerListResource) Configure(_ context.Context, q resource.ConfigureRequest, a *resource.ConfigureResponse) {
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

// The attribute name differs per setting, so values move through
// path-addressed attribute access instead of a fixed struct tag.
func (r *ServerListResource) store(ctx context.Context, state *tfsdk.State, values []string, diags *diag.Diagnostics) {
	if values == nil {
		values = []string{}
	}
	diags.Append(state.SetAttribute(ctx, path.Root("id"), "server")...)
	diags.Append(state.SetAttribute(ctx, path.Root(r.setting.attribute), values)...)
}

func errNotCanonical(want []string) error {
	return fmt.Errorf("write the value in Polyfin's canonical form to keep plans stable: [%s]", strings.Join(want, ", "))
}

func (r *ServerListResource) apply(ctx context.Context, values []string) error {
	want, err := r.setting.canonical(values)
	if err != nil {
		return err
	}
	if !slices.Equal(want, values) {
		return errNotCanonical(want)
	}
	return r.setting.set(r.client, ctx, values)
}

func (r *ServerListResource) Create(ctx context.Context, q resource.CreateRequest, a *resource.CreateResponse) {
	var values []string
	a.Diagnostics.Append(q.Plan.GetAttribute(ctx, path.Root(r.setting.attribute), &values)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	if err := r.apply(ctx, values); err != nil {
		a.Diagnostics.AddError("Polyfin setting refused", err.Error())
		return
	}
	r.store(ctx, &a.State, values, &a.Diagnostics)
}

func (r *ServerListResource) Read(ctx context.Context, q resource.ReadRequest, a *resource.ReadResponse) {
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	current, err := r.setting.get(r.client, ctx)
	if err != nil {
		a.Diagnostics.AddError("Polyfin setting read failed", err.Error())
		return
	}
	r.store(ctx, &a.State, current, &a.Diagnostics)
}

func (r *ServerListResource) Update(ctx context.Context, q resource.UpdateRequest, a *resource.UpdateResponse) {
	var values []string
	a.Diagnostics.Append(q.Plan.GetAttribute(ctx, path.Root(r.setting.attribute), &values)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	if err := r.apply(ctx, values); err != nil {
		a.Diagnostics.AddError("Polyfin setting update refused", err.Error())
		return
	}
	r.store(ctx, &a.State, values, &a.Diagnostics)
}

func (r *ServerListResource) Delete(_ context.Context, _ resource.DeleteRequest, a *resource.DeleteResponse) {
	a.Diagnostics.AddError("Deletion blocked", "The setting remains in Polyfin. Use a reviewed value change; do not reset by destroy.")
}

func (r *ServerListResource) ImportState(ctx context.Context, q resource.ImportStateRequest, a *resource.ImportStateResponse) {
	if q.ID != "server" {
		a.Diagnostics.AddError("Invalid import ID", "Use the singleton ID: server")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, a)
}
