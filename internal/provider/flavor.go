package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	"context"
	"fmt"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"github.com/G-PORTAL/terraform-provider-gpcloud/internal/gpcloudvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &FlavorDataSource{}

func NewFlavor() datasource.DataSource {
	return &FlavorDataSource{}
}

// FlavorDataSource defines the data source implementation.
type FlavorDataSource struct {
	client *client.Client
}

// FlavorDataSourceModel describes the flavor data model.
type FlavorDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	ProjectID    types.String `tfsdk:"project_id"`
	DatacenterID types.String `tfsdk:"datacenter_id"`

	Id types.String `tfsdk:"id"`
}

func (d *FlavorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flavor"
}

func (d *FlavorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The flavor defines the machines hardware specs. Its UUID can be used to choose the hardware to be installed.\n\n" +
			"The name is the name of the Flavor as it appears in the G-PORTAL Cloud Control Panel (example: xeon.2288g.128).\n",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Flavor",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID to consider for flavor availability",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"datacenter_id": schema.StringAttribute{
				MarkdownDescription: "Datacenter ID to consider for flavor availability",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Flavor ID",
			},
		},
	}
}

func (d *FlavorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FlavorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FlavorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	flavorList, err := d.client.CloudClient().ListProjectFlavours(context.Background(), &cloudv1.ListProjectFlavoursRequest{
		Id:           data.ProjectID.ValueString(),
		DatacenterId: data.DatacenterID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error fetching flavor list", err.Error())
		return
	}
	for _, flavor := range flavorList.Flavours {
		if strings.EqualFold(flavor.Name, data.Name.ValueString()) {
			data.Id = types.StringValue(flavor.Id)
			data.Name = types.StringValue(flavor.Name)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}
	resp.Diagnostics.AddError("Flavor not found", fmt.Sprintf("Flavor %s not found for project %s", data.Name.ValueString(), data.ProjectID.ValueString()))
}
