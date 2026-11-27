package provider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// repositoryRoot returns the root of the repository, computed from the
// location of this file, so the checks below never depend on the working
// directory of the test binary.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test source file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// registeredTypeNames returns the provider type name and the type names of
// every resource and data source the provider registers. The provider is the
// only source of truth: a resource is covered as soon as it is registered.
func registeredTypeNames(t *testing.T) (string, []string, []string) {
	t.Helper()
	ctx := context.Background()
	p := New("test")()
	var meta fwprovider.MetadataResponse
	p.Metadata(ctx, fwprovider.MetadataRequest{}, &meta)
	if meta.TypeName == "" {
		t.Fatal("the provider has no type name")
	}
	var resources, dataSources []string
	for _, newResource := range p.Resources(ctx) {
		var resp resource.MetadataResponse
		newResource().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: meta.TypeName}, &resp)
		resources = append(resources, resp.TypeName)
	}
	for _, newDataSource := range p.DataSources(ctx) {
		var resp datasource.MetadataResponse
		newDataSource().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: meta.TypeName}, &resp)
		dataSources = append(dataSources, resp.TypeName)
	}
	return meta.TypeName, resources, dataSources
}

// documentationPage maps a type name to its documentation file name. The
// Terraform Registry strips the provider prefix: polyfin_public_address is
// documented in docs/resources/public_address.md.
func documentationPage(providerName, typeName string) string {
	return strings.TrimPrefix(typeName, providerName+"_")
}

// requireFileContains fails when a file is missing, empty, or does not
// contain the expected snippet.
func requireFileContains(t *testing.T, path, snippet, kind string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s %s is missing: %v", kind, path, err)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		t.Fatalf("%s %s is empty", kind, path)
	}
	if !strings.Contains(string(content), snippet) {
		t.Fatalf("%s %s does not contain %q", kind, path, snippet)
	}
}

// Every registered resource must ship its registry documentation page and its
// example directory. A resource without one of them is unusable from the
// Registry, and nothing else in the build would notice, so the coverage is
// proved here.
func TestRegisteredResourcesHaveADocumentationPageAndAnExample(t *testing.T) {
	root := repositoryRoot(t)
	providerName, resources, _ := registeredTypeNames(t)
	if len(resources) == 0 {
		t.Fatal("the provider registers no resource")
	}
	for _, typeName := range resources {
		t.Run(typeName, func(t *testing.T) {
			requireFileContains(t,
				filepath.Join(root, "docs", "resources", documentationPage(providerName, typeName)+".md"),
				typeName, "documentation page")
			requireFileContains(t,
				filepath.Join(root, "examples", "resources", typeName, "resource.tf"),
				"resource \""+typeName+"\"", "example")
		})
	}
}

// The data sources get the same coverage: a documented data source without an
// example is only half usable from the Registry.
func TestRegisteredDataSourcesHaveADocumentationPageAndAnExample(t *testing.T) {
	root := repositoryRoot(t)
	providerName, _, dataSources := registeredTypeNames(t)
	if len(dataSources) == 0 {
		t.Fatal("the provider registers no data source")
	}
	for _, typeName := range dataSources {
		t.Run(typeName, func(t *testing.T) {
			requireFileContains(t,
				filepath.Join(root, "docs", "data-sources", documentationPage(providerName, typeName)+".md"),
				typeName, "documentation page")
			requireFileContains(t,
				filepath.Join(root, "examples", "data-sources", typeName, "data-source.tf"),
				"data \""+typeName+"\"", "example")
		})
	}
}
