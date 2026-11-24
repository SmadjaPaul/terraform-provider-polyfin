package provider

import (
	"context"

	"github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*UserResource)(nil)
var _ resource.ResourceWithConfigure = (*UserResource)(nil)
var _ resource.ResourceWithImportState = (*UserResource)(nil)

// UserResource manages one local Polyfin account. Every attribute forces a
// replacement: the account is created once and rotating the password means
// recreating it. The password is write-only and never enters the state.
type UserResource struct{ client *polyfin.Client }

type userModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	PasswordWO               types.String `tfsdk:"password_wo"`
	IsAdministrator          types.Bool   `tfsdk:"is_administrator"`
	IsHidden                 types.Bool   `tfsdk:"is_hidden"`
	EnableContentDownloading types.Bool   `tfsdk:"enable_content_downloading"`
}

func NewUserResource() resource.Resource { return &UserResource{} }

func (r *UserResource) Metadata(_ context.Context, q resource.MetadataRequest, a *resource.MetadataResponse) {
	a.TypeName = q.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, a *resource.SchemaResponse) {
	a.Schema = schema.Schema{
		Description: "A local Polyfin account (for example a service account). Does not manage groups, parental control or identity-provider links. Changing the name, password, administrator or hidden flag replaces the account; enable_content_downloading is updated in place.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, Description: "Polyfin user id.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Required: true, Description: "Sign-in name.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"password_wo": schema.StringAttribute{
				Required: true, WriteOnly: true, Sensitive: true,
				Description: "Initial password, sent once at creation and never stored in the state (needs OpenTofu 1.11+ or Terraform 1.11+).",
			},
			"is_administrator": schema.BoolAttribute{Required: true, Description: "Administrator flag. Replaces the account when changed.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"is_hidden":        schema.BoolAttribute{Required: true, Description: "Hide the account from sign-in screens. Replaces the account when changed.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"enable_content_downloading": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "\"Can download\" permission (Jellyfin CanDownload / EnableContentDownloading). Music Assistant marks every track of an account without it as unavailable. Updated in place; only this permission is written, every other setting of the account is left as is.",
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, q resource.ConfigureRequest, a *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, q resource.CreateRequest, a *resource.CreateResponse) {
	var m userModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	var pw types.String
	a.Diagnostics.Append(q.Config.GetAttribute(ctx, path.Root("password_wo"), &pw)...)
	if a.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		a.Diagnostics.AddError("Missing client", "Configure provider")
		return
	}
	u, err := r.client.CreateUser(ctx, m.Name.ValueString(), pw.ValueString(), m.IsAdministrator.ValueBool(), m.IsHidden.ValueBool())
	if err != nil {
		a.Diagnostics.AddError("Polyfin user refused", err.Error())
		return
	}
	if u.Downloads != m.EnableContentDownloading.ValueBool() {
		if u, err = r.client.SetUserDownloads(ctx, u.ID, m.EnableContentDownloading.ValueBool()); err != nil {
			a.Diagnostics.AddError("Polyfin user downloads refused", err.Error())
			return
		}
	}
	m.ID = types.StringValue(u.ID)
	m.PasswordWO = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *UserResource) Read(ctx context.Context, q resource.ReadRequest, a *resource.ReadResponse) {
	var m userModel
	a.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	u, found, err := r.client.FindUser(ctx, m.ID.ValueString())
	if err != nil {
		a.Diagnostics.AddError("Polyfin user read failed", err.Error())
		return
	}
	if !found {
		a.State.RemoveResource(ctx)
		return
	}
	m.Name = types.StringValue(u.Name)
	m.IsAdministrator = types.BoolValue(u.IsAdministrator)
	m.IsHidden = types.BoolValue(u.IsHidden)
	m.EnableContentDownloading = types.BoolValue(u.Downloads)
	m.PasswordWO = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *UserResource) Update(ctx context.Context, q resource.UpdateRequest, a *resource.UpdateResponse) {
	var m userModel
	a.Diagnostics.Append(q.Plan.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SetUserDownloads(ctx, m.ID.ValueString(), m.EnableContentDownloading.ValueBool()); err != nil {
		a.Diagnostics.AddError("Polyfin user downloads refused", err.Error())
		return
	}
	m.PasswordWO = types.StringNull()
	a.Diagnostics.Append(a.State.Set(ctx, &m)...)
}

func (r *UserResource) Delete(ctx context.Context, q resource.DeleteRequest, a *resource.DeleteResponse) {
	var m userModel
	a.Diagnostics.Append(q.State.Get(ctx, &m)...)
	if a.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUser(ctx, m.ID.ValueString()); err != nil {
		a.Diagnostics.AddError("Polyfin user delete failed", err.Error())
	}
}

func (r *UserResource) ImportState(ctx context.Context, q resource.ImportStateRequest, a *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), q, a)
}
