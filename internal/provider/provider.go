package provider

import (
	"context"
	client2 "github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client/auth"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

// Ensure GPCloudProvider satisfies various provider interfaces.
var _ provider.Provider = &GPCloudProvider{}

// GPCloudProvider defines the provider implementation.
type GPCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// GPCloudProviderModel describes the provider data model.
type GPCloudProviderModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Realm        types.String `tfsdk:"realm"`
}

func (p *GPCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gpcloud"
	resp.Version = p.version
}

func (p *GPCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This Terraform Provider uses the [gpcloud-go](https://github.com/G-PORTAL/gpcloud-go) library to interact with the GPCloud API.\n\n" +
			"## Authentication\n" +
			"Authentication will is possible using a Service Account that is created within the GPCloud Panel.\n\n" +
			"When using a service account, you need to provide the `client_id` and `client_secret` which can be created within the GPCloud Panel.\n\n" +
			"The service account behaves like its own user, all actions performed by terraform are made as service account user.\n" +
			"The service account is not able to create / update projects, instead an existing projects need to be manually created and imported using `terraform import` command.\n" +
			"All user-resources (e.g. ssh keys) are created on the service account user.\n",

		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "GRPC Address to connect to",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Client ID",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Client Secret",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "User Email Address",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password",
				Optional:            true,
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Keycloak Realm",
				Optional:            true,
			},
		},
	}
}

func (p *GPCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GPCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	grpcOpts := []interface{}{}
	if !data.Endpoint.IsNull() {
		grpcOpts = append(grpcOpts, client2.EndpointOverrideOption(data.Endpoint.ValueString()))
	}

	realm := "master"
	if !data.Realm.IsNull() {
		realm = data.Realm.ValueString()
	}

	if !data.Username.IsNull() && !data.Password.IsNull() {
		grpcOpts = append(grpcOpts, &auth.ProviderKeycloakUserPassword{
			ClientID:     strings.Trim(data.ClientID.String(), "\""),
			ClientSecret: strings.Trim(data.ClientSecret.String(), "\""),
			Username:     strings.Trim(data.Username.String(), "\""),
			Password:     strings.Trim(data.Password.String(), "\""),
			Realm:        &realm,
		})
	} else {
		grpcOpts = append(grpcOpts, &auth.ProviderKeycloakClientAuth{
			ClientID:     strings.Trim(data.ClientID.String(), "\""),
			ClientSecret: strings.Trim(data.ClientSecret.String(), "\""),
			Realm:        &realm,
		})
	}

	// Example client configuration for data sources and resources
	client, _ := client2.NewClient(grpcOpts...)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *GPCloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProject,
		NewSSHKey,
		NewNode,
		NewProjectImage,
	}
}

func (p *GPCloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFlavour,
		NewImage,
		NewDataCenter,
		NewProjectDS,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GPCloudProvider{
			version: version,
		}
	}
}
