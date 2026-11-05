package polyfin

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestNativeAddonLifecycleWithoutProduction(t *testing.T) {
 const tokenURL="https://example.test/secret-synthetic-token/manifest.json"
 var current Addon
 var calls []string
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls=append(calls,r.Method+" "+r.URL.Path)
  if r.Header.Get("Cookie")!="polyfin_session=synthetic"{t.Error("missing test cookie")}
  w.Header().Set("Content-Type","application/json")
  switch r.URL.Path {
  case "/admin/api/session":
   fmt.Fprint(w,"{\"user\":{\"isAdministrator\":true}}")
  case "/admin/api/scopes/shared/addons":
   if r.Method=="GET"{
    if current.ID==""{fmt.Fprint(w,"[]");return}
    fmt.Fprintf(w,"[{\"id\":\"local-uuid\",\"name\":\"Fixture\",\"kind\":\"stremio\",\"enabled\":%t,\"manifestUrl\":\"REDACTED\"}]",current.Enabled)
    return
   }
   if r.Method=="POST"{
    var input map[string]string
    if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil{t.Error(err)}
    if input["manifestUrl"]!=tokenURL{t.Error("unexpected manifest value")}
    current=Addon{ID:"local-uuid",Name:"Fixture",Kind:"stremio",Enabled:true}
    w.WriteHeader(201)
    json.NewEncoder(w).Encode(current)
    return
   }
  case "/admin/api/scopes/shared/addons/local-uuid":
   if r.Method=="PATCH"{
    var input map[string]any
    if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil{t.Error(err)}
    if v,ok:=input["enabled"];ok{current.Enabled=v.(bool)}
    if v,ok:=input["manifestUrl"];ok && v!=tokenURL {t.Error("unexpected rotation")}
    json.NewEncoder(w).Encode(current)
    return
   }
  }
  http.Error(w,"unknown endpoint",404)
 }))
 defer srv.Close()
 c,err:=New(srv.URL,cookie(t));if err!=nil{t.Fatal(err)}
 if err:=c.CheckAdmin(context.Background());err!=nil{t.Fatal(err)}
 got,err:=c.Install(context.Background(),tokenURL)
 if err!=nil{t.Fatal(err)}
 if got.ID!="local-uuid"||got.Kind!="stremio"{t.Fatalf("unexpected: %+v",got)}
 got,err=c.SetEnabled(context.Background(),got.ID,false)
 if err!=nil||got.Enabled{t.Fatalf("failed disable: %v",err)}
 got,err=c.Replace(context.Background(),got.ID,tokenURL)
 if err!=nil||got.ID!="local-uuid"{t.Fatalf("failed rotation: %v",err)}
 a,found,err:=c.Find(context.Background(),got.ID)
 if err!=nil||!found||a.Enabled{t.Fatalf("failed refresh: %v",err)}
 buf,_:=json.Marshal(a)
 if strings.Contains(string(buf),"REDACTED")||strings.Contains(string(buf),"synthetic-token"){
  t.Fatal("sensitive manifest leaked in model")
 }
 for _,call:=range calls{if strings.HasPrefix(call,"DELETE "){t.Fatal("destructive endpoint called")}}
}
func TestRejectUnsafeLocalAddonIDs(t *testing.T){
 for _,id:=range []string{"../secrets","local-uuid?foo=bar","",strings.Repeat("a",129)}{
  if safeID(id){t.Fatalf("accepted unsafe addon ID %q",id)}
 }
}
