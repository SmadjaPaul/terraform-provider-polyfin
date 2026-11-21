package provider

import (
	"context"

	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*PlaybackSourcePolicyResource)(nil)
var _ resource.ResourceWithConfigure = (*PlaybackSourcePolicyResource)(nil)
var _ resource.ResourceWithImportState = (*PlaybackSourcePolicyResource)(nil)

type PlaybackSourcePolicyResource struct{ client *polyfin.Client }

type playbackSourcePolicyModel struct {
	ID               types.String `tfsdk:"id"`
	SendAppsToSource types.Bool   `tfsdk:"send_apps_to_source"`
}

func NewPlaybackSourcePolicyResource() resource.Resource { return &PlaybackSourcePolicyResource{} }

func (r *PlaybackSourcePolicyResource) Metadata(_ context.Context, q resource.MetadataRequest, a *resource.MetadataResponse) {
	a.TypeName = q.ProviderTypeName + "_playback_source_policy"
}

func (r *PlaybackSourcePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, a *resource.SchemaResponse) {
	a.Schema = schema.Schema{
		Description: "Only Polyfin's server-wide \"Send apps to the source\" playback setting (requires Polyfin 0.83+). Does not change addons, users or other settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Singleton ID: server"},
			"send_apps_to_source": schema.BoolAttribute{
				Required:    true,
				Description: "true redirects every app to the source (the debrid CDN) so video bytes skip the server, and shows the stream address, which may hold addon keys, to every user; false relays non-administrators through the server.",
			},
		},
	}
}

func (r *PlaybackSourcePolicyResource) Configure(_ context.Context, q resource.ConfigureRequest, a *resource.ConfigureResponse) {
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

func (r *PlaybackSourcePolicyResource) Create(ctx context.Context, q resource.CreateRequest, a *resource.CreateResponse) {
	var m playbackSourcePolicyModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	if err := r.client.SetSendAppsToSource(ctx, m.SendAppsToSource.ValueBool()); err != nil {
		a.Diagnostics.AddError("Playback source policy refused", err.Error())
		return
	}
	m.ID = types.StringValue("server")
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *PlaybackSourcePolicyResource) Read(ctx context.Context, q resource.ReadRequest, a *resource.ReadResponse) {
	var m playbackSourcePolicyModel
	a.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	v, err := r.client.GetSendAppsToSource(ctx)
	if err != nil {
		a.Diagnostics.AddError("Playback source policy read failed", err.Error())
		return
	}
	m.ID = types.StringValue("server")
	m.SendAppsToSource = types.BoolValue(v)
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *PlaybackSourcePolicyResource) Update(ctx context.Context, q resource.UpdateRequest, a *resource.UpdateResponse) {
	var m playbackSourcePolicyModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	if err := r.client.SetSendAppsToSource(ctx, m.SendAppsToSource.ValueBool()); err != nil {
		a.Diagnostics.AddError("Playback source policy update refused", err.Error())
		return
	}
	m.ID = types.StringValue("server")
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *PlaybackSourcePolicyResource) Delete(_ context.Context, _ resource.DeleteRequest, a *resource.DeleteResponse) {
	a.Diagnostics.AddError("Deletion blocked", "The playback source policy remains in Polyfin. Use a reviewed value change; do not reset by destroy.")
}

func (r *PlaybackSourcePolicyResource) ImportState(ctx context.Context, q resource.ImportStateRequest, a *resource.ImportStateResponse) {
	if q.ID != "server" {
		a.Diagnostics.AddError("Invalid import ID", "Use the singleton ID: server")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, a)
}
