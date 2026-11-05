package provider

import (
 "context"
 "encoding/json"
 "github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
 "github.com/hashicorp/terraform-plugin-framework/datasource"
 dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
 "github.com/hashicorp/terraform-plugin-framework/types"
)
var _ datasource.DataSource=(*AddonsDataSource)(nil)
var _ datasource.DataSourceWithConfigure=(*AddonsDataSource)(nil)
type AddonsDataSource struct{client *polyfin.Client}
type addonsModel struct{AddonsJSON types.String `tfsdk:"addons_json"`}
func NewAddonsDataSource()datasource.DataSource{return &AddonsDataSource{}}
func(d *AddonsDataSource)Metadata(_ context.Context,q datasource.MetadataRequest,r *datasource.MetadataResponse){r.TypeName=q.ProviderTypeName+"_addons"}
func(d *AddonsDataSource)Schema(_ context.Context,_ datasource.SchemaRequest,r *datasource.SchemaResponse){
 r.Schema=dschema.Schema{Attributes:map[string]dschema.Attribute{
 "addons_json":dschema.StringAttribute{Computed:true,Description:"JSON array of addon id, name, kind, enabled; no manifest URLs."},
 }}
}
func(d *AddonsDataSource)Configure(_ context.Context,q datasource.ConfigureRequest,r *datasource.ConfigureResponse){
 if q.ProviderData==nil{return}
 c,ok:=q.ProviderData.(*polyfin.Client);if !ok{r.Diagnostics.AddError("Invalid provider","Provider client missing");return};d.client=c
}
func(d *AddonsDataSource)Read(ctx context.Context,_ datasource.ReadRequest,r *datasource.ReadResponse){
 if d.client==nil{r.Diagnostics.AddError("Missing client","Configure Polyfin provider.");return}
 if err:=d.client.CheckAdmin(ctx);err!=nil{r.Diagnostics.AddError("Polyfin authorization failed",err.Error());return}
 items,err:=d.client.List(ctx);if err!=nil{r.Diagnostics.AddError("Unable to list addons",err.Error());return}
 // Marshal only the four explicitly safe fields, never the raw API response.
 safe:=make([]polyfin.Addon,0,len(items));safe=append(safe,items...)
 b,err:=json.Marshal(safe);if err!=nil{r.Diagnostics.AddError("Encode failure","Could not serialize addon summaries.");return}
 r.Diagnostics.Append(r.State.Set(ctx,&addonsModel{AddonsJSON:types.StringValue(string(b))})...)
}
