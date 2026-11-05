package main

import (
 "context"
 "log"

 "github.com/SmadjaPaul/terraform-provider-polyfin/internal/provider"
 "github.com/hashicorp/terraform-plugin-framework/providerserver"
)
var version = "dev"
var commit = "unknown"
func main() {
 if err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
  Address: "registry.terraform.io/smadjapaul/polyfin",
 }); err != nil { log.Fatal(err) }
}
