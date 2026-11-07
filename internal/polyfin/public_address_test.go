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

func TestPublicAddressValidation(t *testing.T) {
    for _, a := range []string{"", "http://tv.smadja.dev", "https://tv.smadja.dev/", "https://user:pass@tv.smadja.dev", "https://tv.smadja.dev/path", "https://tv.smadja.dev?x=1", "https://tv.smadja.dev#frag", "https://tv.smadja.dev\n"} {
        if ValidatePublicAddress(a) == nil { t.Errorf("accepted unsafe URL %q", a) }
    }
    if err := ValidatePublicAddress("https://tv.smadja.dev"); err != nil { t.Fatal(err) }
}

func TestPublicAddressPartialUpdatePreservesUnownedSettings(t *testing.T) {
    settings := map[string]any{
        "publicAddress": "https://media.smadja.dev",
        "providerIssuer": "https://auth.smadja.dev/application/o/polyfin/",
        "providerClientId": "polyfin",
        "providerClientSecretSet": true,
        "quickConnectEnabled": true,
        "loginAttempts": float64(5),
    }
    puts := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
        if r.Header.Get("Cookie") != "polyfin_session=synthetic" { t.Error("missing synthetic admin cookie") }
        w.Header().Set("Content-Type", "application/json")
        switch {
        case r.URL.Path == "/admin/api/session" && r.Method == "GET":
            fmt.Fprint(w,`{"user":{"isAdministrator":true}}`)
        case r.URL.Path == "/admin/api/settings" && r.Method == "GET":
            _ = json.NewEncoder(w).Encode(settings)
        case r.URL.Path == "/admin/api/settings" && r.Method == "PUT":
            puts++
            var data map[string]any
            if err := json.NewDecoder(r.Body).Decode(&data);err!=nil { t.Fatal(err) }
            if len(data)!=1 || data["publicAddress"]!="https://tv.smadja.dev" { t.Errorf("unexpected PUT keys: %v",data) }
            for k,v := range data { settings[k]=v }
            _ = json.NewEncoder(w).Encode(settings)
        default:
            http.Error(w,"not found",404)
        }
    }))
    defer srv.Close()
    c,err:=New(srv.URL,cookie(t))
    if err!=nil { t.Fatal(err) }
    if err:=c.SetPublicAddress(context.Background(),"https://tv.smadja.dev");err!=nil { t.Fatal(err) }
    if puts!=1 { t.Fatalf("expected one non-destructive PUT, got %d",puts) }
    if settings["providerIssuer"]!="https://auth.smadja.dev/application/o/polyfin/" || settings["providerClientSecretSet"]!=true || settings["quickConnectEnabled"]!=true || settings["loginAttempts"]!=float64(5) {
        t.Fatal("unowned settings changed")
    }
    got,err:=c.GetPublicAddress(context.Background())
    if err!=nil || got!="https://tv.smadja.dev" { t.Fatalf("read after update: %q, %v",got,err) }
    if strings.Contains(fmt.Sprint(settings),"synthetic-password") { t.Fatal("unexpected secret in test state") }
}

func TestPublicAddressRejectsNonAdmin(t *testing.T) {
    puts:=0
    srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        w.Header().Set("Content-Type","application/json")
        if r.URL.Path=="/admin/api/session" { fmt.Fprint(w,`{"user":{"isAdministrator":false}}`);return }
        puts++;http.Error(w,"forbidden",403)
    }))
    defer srv.Close()
    c,err:=New(srv.URL,cookie(t));if err!=nil{t.Fatal(err)}
    if err:=c.SetPublicAddress(context.Background(),"https://tv.smadja.dev");err==nil {t.Fatal("non-admin accepted")}
    if puts!=0 {t.Fatal("non-admin reached the settings endpoint")}
}
