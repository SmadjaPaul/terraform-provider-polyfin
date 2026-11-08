package provider

import (
 "context"
 "os"
 "github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
 "github.com/hashicorp/terraform-plugin-framework/datasource"
 "github.com/hashicorp/terraform-plugin-framework/provider"
 schema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
 "github.com/hashicorp/terraform-plugin-framework/resource"
 "github.com/hashicorp/terraform-plugin-framework/types"
)
var _ provider.Provider = (*PolyfinProvider)(nil)
type PolyfinProvider struct{version string}
type model struct{Endpoint types.String `tfsdk:"endpoint"`}
func New(version string)func()provider.Provider{return func()provider.Provider{return &PolyfinProvider{version:version}}}
func(p *PolyfinProvider)Metadata(_ context.Context,_ provider.MetadataRequest,r *provider.MetadataResponse){r.TypeName="polyfin";r.Version=p.version}
func(p *PolyfinProvider)Schema(_ context.Context,_ provider.SchemaRequest,r *provider.SchemaResponse){
 r.Schema=schema.Schema{Attributes:map[string]schema.Attribute{
 "endpoint":schema.StringAttribute{Optional:true,Description:"Polyfin origin; HTTPS required outside loopback/in-cluster Service."},
 }}
}
func(p *PolyfinProvider)Configure(ctx context.Context,q provider.ConfigureRequest,r *provider.ConfigureResponse){
 var m model;r.Diagnostics.Append(q.Config.Get(ctx,&m)...);if r.Diagnostics.HasError(){return}
 endpoint:=m.Endpoint.ValueString();if endpoint==""{endpoint=os.Getenv("POLYFIN_ENDPOINT")}
 if endpoint==""{r.Diagnostics.AddError("Missing endpoint","Set endpoint or POLYFIN_ENDPOINT.");return}
 c,err:=polyfin.NewFromEnvironment(endpoint)
 if err!=nil{r.Diagnostics.AddError("Polyfin provider configuration failed",err.Error());return}
 r.ResourceData=c;r.DataSourceData=c
}
func(p *PolyfinProvider)Resources(_ context.Context)[]func()resource.Resource{
 return []func()resource.Resource{
  func()resource.Resource{return NewAddonResource()},
  func()resource.Resource{return NewPublicAddressResource()},
  func()resource.Resource{return NewPlaybackLanguagesResource()},
 }
}
func(p *PolyfinProvider)DataSources(_ context.Context)[]func()datasource.DataSource{
 return []func()datasource.DataSource{func()datasource.DataSource{return NewAddonsDataSource()}}
}
