package polyfin

import (
 "context"
 "errors"
 "fmt"
)

type PlaybackLanguages struct {
 Audio string `json:"defaultAudioLanguage"`
 Subtitles string `json:"defaultSubtitleLanguage"`
}

// These are ISO 639-2/B codes, not ISO 639-1 language tags.
// Verify the accepted values against upstream's TrackLanguages on every write.
func (c *Client) GetPlaybackLanguages(ctx context.Context) (PlaybackLanguages,error) {
 if err:=c.CheckAdmin(ctx);err!=nil{return PlaybackLanguages{},err}
 var result struct {
  Audio *string `json:"defaultAudioLanguage"`
  Subtitles *string `json:"defaultSubtitleLanguage"`
 }
 _,err:=c.request(ctx,"GET","/admin/api/settings",nil,&result)
 if err!=nil{return PlaybackLanguages{},err}
 if result.Audio==nil||result.Subtitles==nil{return PlaybackLanguages{},errors.New("Polyfin settings missing default language properties")}
 return PlaybackLanguages{Audio:*result.Audio,Subtitles:*result.Subtitles},nil
}

func (c *Client) SetPlaybackLanguages(ctx context.Context, languages PlaybackLanguages) error {
 // Bound the resource to the specific safe language codes needed by the
 // family. Empty is the upstream-supported "no preference" option.
 allowed:=map[string]bool{"":true,"fre":true,"eng":true,"spa":true,"jpn":true}
 if !allowed[languages.Audio]||!allowed[languages.Subtitles]{
  return errors.New("unsupported default language: use ISO 639-2/B fre, eng, spa, jpn or empty")
 }
 if err:=c.CheckAdmin(ctx);err!=nil{return err}
 _,err:=c.request(ctx,"PUT","/admin/api/settings",map[string]string{
   "defaultAudioLanguage":languages.Audio,
   "defaultSubtitleLanguage":languages.Subtitles,
 },nil)
 if err!=nil{return err}
 current,err:=c.GetPlaybackLanguages(ctx)
 if err!=nil{return err}
 if current!=languages{return fmt.Errorf("Polyfin playback default language readback mismatch (values suppressed)")}
 return nil
}
