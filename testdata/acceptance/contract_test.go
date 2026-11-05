package acceptance

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)

// Static, non-network checks only. They do not imply runtime acceptance.
func TestComposeIsolationContract(t *testing.T) {
 b,err:=os.ReadFile(filepath.Join("compose.yaml"))
 if err!=nil{t.Fatal(err)}
 s:=string(b)
 for _,required:=range []string{
  "name: polyfin-provider-acceptance",
  "127.0.0.1:18096:8096",
  "polyfin:0.60.0@sha256:e2b1d1624a985292ef7a69434a9fd43cea2ead37202b1860c659d18a609125a6",
  "postgres:18.6@sha256:5a5a84b19854a9ffaa54082c166ff4ec27473a361e496e5ea167f298f2da9722",
  "internal: true",
  "ACCEPTANCE_POSTGRES_PASSWORD",
  "ACCEPTANCE_POLYFIN_SECRET_KEY",
  "read_only: true",
 }{if !strings.Contains(s,required){t.Errorf("missing required isolation setting: %s",required)}}
 for _,forbidden:=range []string{
  "media.smadja.dev","aiostreams.smadja.dev","auth.smadja.dev",
  "hostNetwork: true","network_mode: host","/var/run/docker.sock",
  "5432:5432","POLYFIN_OIDC", "external: true",
 }{if strings.Contains(s,forbidden){t.Errorf("unexpected production or host exposure: %s",forbidden)}}
}
