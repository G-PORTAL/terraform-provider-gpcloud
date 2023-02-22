package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	"context"
	"fmt"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"github.com/G-PORTAL/terraform-provider-gpcloud/internal/gpcloudvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ImageDataSource{}

func NewImage() datasource.DataSource {
	return &ImageDataSource{}
}

// ImageDataSource defines the data source implementation.
type ImageDataSource struct {
	client *client.Client
}

// ImageDataSourceModel describes the data source data model.
type ImageDataSourceModel struct {
	Name                types.String `tfsdk:"name"`
	FlavourID           types.String `tfsdk:"flavour_id"`
	AuthenticationTypes types.List   `tfsdk:"authentication_types"`

	Id types.String `tfsdk:"id"`
}

func (d *ImageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *ImageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "The datasource image represents the cloud images provided by GPCloud.\n\n" +
			"The resulting UUID can be used to provision nodes.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the image",
				Required:            true,
			},
			"flavour_id": schema.StringAttribute{
				MarkdownDescription: "Flavour ID",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"authentication_types": schema.ListAttribute{
				MarkdownDescription: "List of valid authentication methods",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Image ID",
			},
		},
	}
}

func (d *ImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImageDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	imageList, err := d.client.CloudClient().ListPublicImages(context.Background(), &cloudv1.ListPublicImagesRequest{
		FlavourId: data.FlavourID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error while fetching image list", err.Error())
	}
	for _, image := range imageList.Images {
		if image.Name == data.Name.ValueString() {
			data.Id = types.StringValue(image.Id)
			data.Name = types.StringValue(image.Name)
			authTypes := make([]attr.Value, 0)
			for _, authType := range image.AuthenticationTypes {
				authTypes = append(authTypes, types.StringValue(authType.String()))
			}
			data.AuthenticationTypes, _ = types.ListValue(types.StringType, authTypes)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}
	resp.Diagnostics.AddError("Image not found", fmt.Sprintf("Image with name %s not found", data.Name.ValueString()))
}
