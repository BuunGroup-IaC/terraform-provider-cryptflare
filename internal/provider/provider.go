// Copyright (c) 2026 Buun Group
// SPDX-License-Identifier: MPL-2.0

// Package provider implements the CryptFlare Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/buun-group/terraform-provider-cryptflare/internal/client"
	"github.com/buun-group/terraform-provider-cryptflare/internal/service/environment"
	"github.com/buun-group/terraform-provider-cryptflare/internal/service/pod"
	"github.com/buun-group/terraform-provider-cryptflare/internal/service/secret"
	"github.com/buun-group/terraform-provider-cryptflare/internal/service/workspace"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &CryptFlareProvider{}

// CryptFlareProvider implements the CryptFlare Terraform provider.
type CryptFlareProvider struct {
	version string
}

// CryptFlareProviderModel describes the provider configuration.
type CryptFlareProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	APIURL   types.String `tfsdk:"api_url"`
	OrgID    types.String `tfsdk:"org_id"`
}

// New returns a provider.Provider constructor function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CryptFlareProvider{version: version}
	}
}

// Metadata returns the provider type name.
func (p *CryptFlareProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cryptflare"
	resp.Version = p.version
}

// Schema defines the provider configuration attributes.
func (p *CryptFlareProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage CryptFlare secrets, workspaces, environments, and pods as infrastructure-as-code.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "CryptFlare API token. Can also be set via the CF_TOKEN environment variable.",
			},
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "CryptFlare API base URL. Defaults to https://api.cryptflare.com. Can also be set via CF_API_URL.",
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Description: "Default organisation ID. Can also be set via CF_ORG.",
			},
		},
	}
}

// Configure creates the API client from provider config.
func (p *CryptFlareProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config CryptFlareProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve values: config > env > default
	apiToken := resolveString(config.APIToken, "CF_TOKEN", "")
	apiURL := resolveString(config.APIURL, "CF_API_URL", "")
	orgID := resolveString(config.OrgID, "CF_ORG", "")

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"Set api_token in the provider block or the CF_TOKEN environment variable.",
		)
		return
	}

	if orgID == "" {
		resp.Diagnostics.AddError(
			"Missing organisation ID",
			"Set org_id in the provider block or the CF_ORG environment variable.",
		)
		return
	}

	c := client.New(apiURL, apiToken, orgID)

	resp.DataSourceData = c
	resp.ResourceData = c
}

// Resources returns the provider's resources.
func (p *CryptFlareProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		workspace.NewResource,
		environment.NewResource,
		secret.NewResource,
		pod.NewResource,
	}
}

// DataSources returns the provider's data sources.
func (p *CryptFlareProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func resolveString(tfValue types.String, envVar, defaultValue string) string {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueString()
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}
