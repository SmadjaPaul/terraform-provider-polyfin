package polyfin

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/url"
 "os"
 "strings"
 "sync"
 "time"
)

type Addon struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Kind string `json:"kind"`
 Enabled bool `json:"enabled"`
}
type Client struct {
 endpoint string
 cookie string
 http *http.Client
 loginName string
 loginPassword string
 loginMu sync.Mutex
}
func New(endpoint, cookieFile string) (*Client,error) {
 u,err:=url.Parse(endpoint)
 if err!=nil || u.Host=="" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" || u.Path!="" {
  return nil,errors.New("Polyfin endpoint must be a bare origin without credentials or query")
 }
 host:=strings.ToLower(u.Hostname())
 if u.Scheme!="https" && !(u.Scheme=="http" && (host=="localhost" || host=="127.0.0.1" || host=="::1" || strings.HasSuffix(host,".svc.cluster.local"))) {
  return nil,errors.New("Polyfin endpoint requires HTTPS, except for loopback or Kubernetes Service")
 }
 if cookieFile=="" { return nil,errors.New("POLYFIN_ADMIN_COOKIE_FILE is required") }
 b,err:=os.ReadFile(cookieFile)
 if err!=nil {return nil,errors.New("unable to read Polyfin admin cookie file (path suppressed)")}
 cookie:=strings.TrimSpace(string(b))
 if cookie=="" || strings.ContainsAny(cookie,"\r\n") {return nil,errors.New("invalid Polyfin cookie file")}
 // Never send the administrator cookie to a redirect destination.
 client:=&http.Client{Timeout:20*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 return &Client{endpoint:strings.TrimRight(endpoint,"/"),cookie:cookie,http:client},nil
}
// NewFromEnvironment supports an Atlantis-safe noninteractive native
// administrator login. Neither passwords nor cookies live in Terraform state.
func NewFromEnvironment(endpoint string)(*Client,error){
 cookieFile:=os.Getenv("POLYFIN_ADMIN_COOKIE_FILE")
 nameFile:=os.Getenv("POLYFIN_ADMIN_USERNAME_FILE")
 passwordFile:=os.Getenv("POLYFIN_ADMIN_PASSWORD_FILE")
 if cookieFile!="" && (nameFile!=""||passwordFile!=""){
  return nil,errors.New("ambiguous Polyfin admin authentication modes")
 }
 if cookieFile!=""{return New(endpoint,cookieFile)}
 if nameFile==""||passwordFile==""{return nil,errors.New("set cookie file or both admin username/password files")}
 nameBytes,nameErr:=os.ReadFile(nameFile)
 passBytes,passErr:=os.ReadFile(passwordFile)
 if nameErr!=nil||passErr!=nil{return nil,errors.New("unable to read Polyfin administrator credential files")}
 name:=strings.TrimSpace(string(nameBytes))
 pass:=strings.TrimRight(string(passBytes),"\r\n")
 if name==""||pass==""{return nil,errors.New("empty Polyfin administrator credentials")}
 // Reuse the same endpoint/transport validation, with no source cookie file.
 c,err:=NewForLogin(endpoint)
 if err!=nil{return nil,err}
 c.loginName=name;c.loginPassword=pass
 return c,nil
}
func NewForLogin(endpoint string)(*Client,error) {
 // Cookie-less client uses the same validated origin and redirect policy.
 u,err:=url.Parse(endpoint)
 if err!=nil||u.Host==""||u.User!=nil||u.RawQuery!=""||u.Fragment!=""||u.Path!=""{
  return nil,errors.New("Polyfin endpoint must be a bare origin without credentials or query")
 }
 host:=strings.ToLower(u.Hostname())
 if u.Scheme!="https" && !(u.Scheme=="http" && (host=="localhost"||host=="127.0.0.1"||host=="::1"||strings.HasSuffix(host,".svc.cluster.local"))){
  return nil,errors.New("Polyfin endpoint requires HTTPS, except for loopback or Kubernetes Service")
 }
 httpClient:=&http.Client{Timeout:20*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 return &Client{endpoint:strings.TrimRight(endpoint,"/"),http:httpClient},nil
}
func (c *Client) ensureSession(ctx context.Context)error{
 c.loginMu.Lock()
 defer c.loginMu.Unlock()
 if c.cookie!=""{return nil}
 if c.loginName==""||c.loginPassword==""{return errors.New("Polyfin admin authentication unavailable")}
 // Login response contains a session cookie, not a bearer token.
 payload:=map[string]string{"name":c.loginName,"password":c.loginPassword}
 b,_:=json.Marshal(payload)
 req,err:=http.NewRequestWithContext(ctx,"POST",c.endpoint+"/admin/api/session",bytes.NewReader(b))
 if err!=nil{return errors.New("Polyfin login request failed")}
 req.Header.Set("Content-Type","application/json")
 res,err:=c.http.Do(req)
 if err!=nil{return errors.New("Polyfin login transport failed")}
 defer res.Body.Close()
 if res.StatusCode<200||res.StatusCode>=300{
  return fmt.Errorf("Polyfin login returned HTTP %d (body suppressed)",res.StatusCode)
 }
 for _,cookie:=range res.Cookies(){
  if cookie.Name=="polyfin_session" && cookie.Value!=""{
   c.cookie="polyfin_session="+cookie.Value
   return nil
  }
 }
 return errors.New("Polyfin login returned no session cookie")
}
func (c *Client) request(ctx context.Context,method,path string,body any,out any) (int,error) {
 var reader io.Reader
 if body!=nil {
  b,err:=json.Marshal(body);if err!=nil{return 0,errors.New("could not encode request")}
  reader=bytes.NewReader(b)
 }
 req,err:=http.NewRequestWithContext(ctx,method,c.endpoint+path,reader)
 if err!=nil{return 0,errors.New("could not build Polyfin API request")}
 req.Header.Set("Cookie",c.cookie);req.Header.Set("Accept","application/json")
 if body!=nil{req.Header.Set("Content-Type","application/json")}
 res,err:=c.http.Do(req)
 if err!=nil {
  // Never forward underlying network errors: they may contain URL/credentials.
  return 0,errors.New("Polyfin API connection failed (details suppressed)")
 }
 defer res.Body.Close()
 if res.StatusCode<200||res.StatusCode>=300{
  _,_=io.Copy(io.Discard,io.LimitReader(res.Body,4096))
  return res.StatusCode,fmt.Errorf("Polyfin API %s returned HTTP %d (body suppressed)",method,res.StatusCode)
 }
 if out!=nil {
  dec:=json.NewDecoder(io.LimitReader(res.Body,2<<20))
  if err:=dec.Decode(out);err!=nil{return res.StatusCode,errors.New("invalid Polyfin JSON response")}
 }
 return res.StatusCode,nil
}
func (c *Client) CheckAdmin(ctx context.Context)error{
 if err:=c.ensureSession(ctx);err!=nil{return err}
 var session struct{
 User struct{IsAdministrator bool `json:"isAdministrator"`} `json:"user"`
}
 _,err:=c.request(ctx,"GET","/admin/api/session",nil,&session)
 if err!=nil{return err}
 if !session.User.IsAdministrator{return errors.New("Polyfin session is not administrator")}
 return nil
}
func (c *Client) List(ctx context.Context)([]Addon,error){
 var out []Addon
 _,err:=c.request(ctx,"GET","/admin/api/scopes/shared/addons",nil,&out)
 return out,err
}
func (c *Client) Find(ctx context.Context,id string)(Addon,bool,error){
 addons,err:=c.List(ctx);if err!=nil{return Addon{},false,err}
 for _,a:=range addons{if a.ID==id{return a,true,nil}}
 return Addon{},false,nil
}
func (c *Client) Install(ctx context.Context,manifestURL string)(Addon,error){
 var out Addon
 _,err:=c.request(ctx,"POST","/admin/api/scopes/shared/addons",map[string]string{"manifestUrl":manifestURL},&out)
 if err!=nil{return Addon{},err}
 if out.ID=="" || (out.Kind!="stremio" && out.Kind!="eclipse"){return Addon{},errors.New("Polyfin returned invalid addon identity")}
 return out,nil
}
func (c *Client) SetEnabled(ctx context.Context,id string,enabled bool)(Addon,error){
 var out Addon
 if !safeID(id){return Addon{},errors.New("invalid addon identifier")}
 _,err:=c.request(ctx,"PATCH","/admin/api/scopes/shared/addons/"+url.PathEscape(id),map[string]bool{"enabled":enabled},&out)
 return out,err
}
func (c *Client) Replace(ctx context.Context,id,manifestURL string)(Addon,error){
 var out Addon
 if !safeID(id){return Addon{},errors.New("invalid addon identifier")}
 _,err:=c.request(ctx,"PATCH","/admin/api/scopes/shared/addons/"+url.PathEscape(id),map[string]string{"manifestUrl":manifestURL},&out)
 return out,err
}
func safeID(id string)bool {
 if len(id)==0||len(id)>128{return false}
 for _,r:=range id{if !(r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='-'){return false}}
 return true
}
func HostIsLiteralIP(s string)bool{return net.ParseIP(s)!=nil}
