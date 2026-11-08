package provider

import (
 "context"
 "github.com/SmadjaPaul/terraform-provider-polyfin/internal/polyfin"
 "github.com/hashicorp/terraform-plugin-framework/path"
 "github.com/hashicorp/terraform-plugin-framework/resource"
 schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
 "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*PlaybackLanguagesResource)(nil)
var _ resource.ResourceWithConfigure = (*PlaybackLanguagesResource)(nil)
var _ resource.ResourceWithImportState = (*PlaybackLanguagesResource)(nil)

type PlaybackLanguagesResource struct{client *polyfin.Client}
type playbackLanguagesModel struct{
 ID types.String `tfsdk:"id"`
 Audio types.String `tfsdk:"default_audio_language"`
 Subtitles types.String `tfsdk:"default_subtitle_language"`
}
func NewPlaybackLanguagesResource() resource.Resource{return &PlaybackLanguagesResource{}}
func(r *PlaybackLanguagesResource)Metadata(_ context.Context,q resource.MetadataRequest,a *resource.MetadataResponse){
 a.TypeName=q.ProviderTypeName+"_playback_languages"
}
func(r *PlaybackLanguagesResource)Schema(_ context.Context,_ resource.SchemaRequest,a *resource.SchemaResponse){
 a.Schema=schema.Schema{
  Description:"Only Polyfin server-wide default track languages. Does not change user-level choices or stream sources.",
  Attributes:map[string]schema.Attribute{
   "id":schema.StringAttribute{Computed:true,Description:"Singleton ID: server"},
   "default_audio_language":schema.StringAttribute{Required:true,Description:"ISO 639-2/B: fre, eng, spa, jpn, or empty."},
   "default_subtitle_language":schema.StringAttribute{Required:true,Description:"ISO 639-2/B: fre, eng, spa, jpn, or empty."},
  },
 }
}
func(r *PlaybackLanguagesResource)Configure(_ context.Context,q resource.ConfigureRequest,a *resource.ConfigureResponse){
 if q.ProviderData==nil{return}
 c,ok:=q.ProviderData.(*polyfin.Client)
 if !ok{a.Diagnostics.AddError("Invalid provider","Polyfin client missing");return}
 r.client=c
}
func(r *PlaybackLanguagesResource)Create(ctx context.Context,q resource.CreateRequest,a *resource.CreateResponse){
 var m playbackLanguagesModel;a.Diagnostics.Append(q.Plan.Get(ctx,&m)...);if a.Diagnostics.HasError(){return}
 if r.client==nil{a.Diagnostics.AddError("Missing client","Configure provider");return}
 if err:=r.client.SetPlaybackLanguages(ctx,polyfin.PlaybackLanguages{Audio:m.Audio.ValueString(),Subtitles:m.Subtitles.ValueString()});err!=nil{
  a.Diagnostics.AddError("Playback defaults refused",err.Error());return
 }
 m.ID=types.StringValue("server");a.Diagnostics.Append(a.State.Set(ctx,&m)...)
}
func(r *PlaybackLanguagesResource)Read(ctx context.Context,q resource.ReadRequest,a *resource.ReadResponse){
 var m playbackLanguagesModel;a.Diagnostics.Append(q.State.Get(ctx,&m)...);if a.Diagnostics.HasError(){return}
 if r.client==nil{a.Diagnostics.AddError("Missing client","Configure provider");return}
 x,err:=r.client.GetPlaybackLanguages(ctx);if err!=nil{a.Diagnostics.AddError("Playback defaults read failed",err.Error());return}
 m.ID=types.StringValue("server");m.Audio=types.StringValue(x.Audio);m.Subtitles=types.StringValue(x.Subtitles)
 a.Diagnostics.Append(a.State.Set(ctx,&m)...)
}
func(r *PlaybackLanguagesResource)Update(ctx context.Context,q resource.UpdateRequest,a *resource.UpdateResponse){
 var m playbackLanguagesModel;a.Diagnostics.Append(q.Plan.Get(ctx,&m)...);if a.Diagnostics.HasError(){return}
 if r.client==nil{a.Diagnostics.AddError("Missing client","Configure provider");return}
 if err:=r.client.SetPlaybackLanguages(ctx,polyfin.PlaybackLanguages{Audio:m.Audio.ValueString(),Subtitles:m.Subtitles.ValueString()});err!=nil{
  a.Diagnostics.AddError("Playback defaults update refused",err.Error());return
 }
 m.ID=types.StringValue("server");a.Diagnostics.Append(a.State.Set(ctx,&m)...)
}
func(r *PlaybackLanguagesResource)Delete(_ context.Context,_ resource.DeleteRequest,a *resource.DeleteResponse){
 a.Diagnostics.AddError("Deletion blocked","Playback defaults remain in Polyfin. Use a reviewed value change; do not reset by destroy.")
}
func(r *PlaybackLanguagesResource)ImportState(ctx context.Context,q resource.ImportStateRequest,a *resource.ImportStateResponse){
 if q.ID!="server"{a.Diagnostics.AddError("Invalid import ID","Use the singleton ID: server");return}
 resource.ImportStatePassthroughID(ctx,path.Root("id"),q,a)
}
