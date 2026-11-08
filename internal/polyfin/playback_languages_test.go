package polyfin

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "testing"
)

func TestPlaybackLanguagesPartialSettings(t *testing.T){
 settings:=map[string]any{
  "defaultAudioLanguage":"","defaultSubtitleLanguage":"",
  "publicAddress":"https://tv.smadja.dev",
  "providerIssuer":"https://auth.smadja.dev/application/o/polyfin/",
  "providerClientSecretSet":true,
  "versionOrder":[]string{"resolution-high","source"},
 }
 puts:=0
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  if r.Header.Get("Cookie")!="polyfin_session=synthetic"{t.Error("cookie not set")}
  switch{
  case r.URL.Path=="/admin/api/session"&&r.Method=="GET":
   fmt.Fprint(w,`{"user":{"isAdministrator":true}}`)
  case r.URL.Path=="/admin/api/settings"&&r.Method=="GET":
   _=json.NewEncoder(w).Encode(settings)
  case r.URL.Path=="/admin/api/settings"&&r.Method=="PUT":
   puts++
   var data map[string]string
   if err:=json.NewDecoder(r.Body).Decode(&data);err!=nil{t.Error(err)}
   if len(data)!=2||data["defaultAudioLanguage"]!="fre"||data["defaultSubtitleLanguage"]!="fre"{t.Errorf("unwanted settings update: %v",data)}
   for k,v:=range data{settings[k]=v}
   _=json.NewEncoder(w).Encode(settings)
  default:http.Error(w,"not found",404)
  }
 }))
 defer srv.Close()
 c,err:=New(srv.URL,cookie(t));if err!=nil{t.Fatal(err)}
 if err:=c.SetPlaybackLanguages(context.Background(),PlaybackLanguages{Audio:"fre",Subtitles:"fre"});err!=nil{t.Fatal(err)}
 if puts!=1{t.Fatalf("expected 1 partial PUT, got %d",puts)}
 if settings["publicAddress"]!="https://tv.smadja.dev"||settings["providerClientSecretSet"]!=true{t.Fatal("OIDC settings changed")}
 if got,err:=c.GetPlaybackLanguages(context.Background());err!=nil||got.Audio!="fre"||got.Subtitles!="fre"{t.Fatalf("readback %#v %v",got,err)}
 if err:=c.SetPlaybackLanguages(context.Background(),PlaybackLanguages{Audio:"rus",Subtitles:"fre"});err==nil{t.Fatal("invalid language accepted")}
 if puts!=1{t.Fatal("invalid language reached PUT endpoint")}
}
