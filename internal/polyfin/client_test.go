package polyfin

import (
 "context"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func cookie(t *testing.T)string{
 t.Helper();p:=filepath.Join(t.TempDir(),"cookie");if err:=os.WriteFile(p,[]byte("polyfin_session=synthetic"),0600);err!=nil{t.Fatal(err)};return p
}
func TestEndpointValidation(t *testing.T){
 for _,s:=range []string{"http://example.com","https://user:password@example.com","https://example.com/path","https://example.com?token=oops","file:///etc/passwd"}{
  if _,err:=New(s,cookie(t));err==nil{t.Fatalf("accepted %s",s)}
 }
}
func TestAdminAndAddonSafeRead(t *testing.T){
 var seen []string
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Cookie")!="polyfin_session=synthetic"{t.Error("cookie not provided")}
  seen=append(seen,r.URL.Path)
  switch r.URL.Path{
  case "/admin/api/session":w.Write([]byte(`{"isAdministrator":true}`))
  case "/admin/api/scopes/shared/addons":w.Write([]byte(`[{"id":"local-uuid","name":"Nebula","kind":"eclipse","enabled":true,"manifestUrl":"SECRET_DO_NOT_EMIT"}]`))
  default:t.Error("unexpected path")
  }
 }))
 defer srv.Close()
 c,e:=New(srv.URL,cookie(t));if e!=nil{t.Fatal(e)}
 if err:=c.CheckAdmin(context.Background());err!=nil{t.Fatal(err)}
 list,err:=c.List(context.Background());if err!=nil{t.Fatal(err)}
 if len(list)!=1||list[0].ID!="local-uuid"{t.Fatalf("unexpected addons %+v",list)}
 if strings.Contains(list[0].Name,"SECRET_DO_NOT_EMIT"){t.Fatal("secret exposed")}
 if len(seen)!=2{t.Fatal(seen)}
}
func TestRedirectRefused(t *testing.T){
 srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Redirect(w,r,"https://example.com",302)}));defer srv.Close()
 c,err:=New(srv.URL,cookie(t));if err!=nil{t.Fatal(err)}
 if _,err:=c.List(context.Background());err==nil{t.Fatal("accepted redirect")}
}
