package provider

import (
"context"
"strings"
"testing"

"github.com/hashicorp/terraform-plugin-framework/datasource"
"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestPreviewAPIDisclaimers ensures that all resources and data sources that use
// a preview API (detected by checking the client code for "-preview" version strings)
// include a proper preview disclaimer in their MarkdownDescription.
//
// This test helps enforce consistent documentation for preview resources so that
// users are clearly informed about the instability of preview APIs.
// See: https://docs.meshcloud.io/api/technical-specifications#preview-endpoints
func TestPreviewAPIDisclaimers(t *testing.T) {
previewResources := []string{
"meshstack_building_block_definition",
"meshstack_building_block_v2",
"meshstack_tenant_v4",
}
previewDataSources := []string{
"meshstack_building_block_v2",
"meshstack_tenant_v4",
}

providerImpl := &MeshStackProvider{version: "test"}

resourceImpls := providerImpl.Resources(context.Background())
dataSourceImpls := providerImpl.DataSources(context.Background())

for _, resourceName := range previewResources {
found := false
for _, r := range resourceImpls {
res := r()
var meta resource.MetadataResponse
res.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "meshstack"}, &meta)
if meta.TypeName != resourceName {
continue
}
found = true

var schemaResp resource.SchemaResponse
res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
desc := schemaResp.Schema.MarkdownDescription
if !strings.Contains(desc, "preview-endpoints") {
t.Errorf("Resource %s uses a preview API but its MarkdownDescription does not include a link to preview-endpoints documentation. "+
"Add a preview disclaimer linking to https://docs.meshcloud.io/api/technical-specifications#preview-endpoints", resourceName)
}
}
if !found {
t.Errorf("Resource %s not found in provider resources list", resourceName)
}
}

for _, dsName := range previewDataSources {
found := false
for _, d := range dataSourceImpls {
ds := d()
var meta datasource.MetadataResponse
ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "meshstack"}, &meta)
if meta.TypeName != dsName {
continue
}
found = true

var schemaResp datasource.SchemaResponse
ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
desc := schemaResp.Schema.MarkdownDescription
if !strings.Contains(desc, "preview-endpoints") {
t.Errorf("DataSource %s uses a preview API but its MarkdownDescription does not include a link to preview-endpoints documentation. "+
"Add a preview disclaimer linking to https://docs.meshcloud.io/api/technical-specifications#preview-endpoints", dsName)
}
}
if !found {
t.Errorf("DataSource %s not found in provider data sources list", dsName)
}
}
}
