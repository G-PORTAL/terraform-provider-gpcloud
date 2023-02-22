package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	"context"
	"fmt"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataCenterDataSource{}

func NewDataCenter() datasource.DataSource {
	return &DataCenterDataSource{}
}

// DataCenterDataSource defines the data source implementation.
type DataCenterDataSource struct {
	client *client.Client
}

// DataCenterDataSourceModel describes the DataCEnter data model.
type DataCenterDataSourceModel struct {
	Name            types.String `tfsdk:"name"`
	Short           types.String `tfsdk:"short"`
	RegionID        types.String `tfsdk:"region_id"`
	LatencyEndpoint types.String `tfsdk:"latency_endpoint"`
	ServerPrefix    types.String `tfsdk:"server_prefix"`

	Id types.String `tfsdk:"id"`
}

func (d *DataCenterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datacenter"
}

func (d *DataCenterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Datacenter is a virtual resource that allows fetching the Datacenter UUID by defining the Datacenter short name.\n\n" +
			"The short name is the short of the Datacenter as it appears in the G-PORTAL Cloud Control Panel (example: fra01).\n",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the datacenter",
				Computed:            true,
			},
			"short": schema.StringAttribute{
				MarkdownDescription: "Short name of the datacenter",
				Required:            true,
			},
			"region_id": schema.StringAttribute{
				MarkdownDescription: "Region ID of the datacenter",
				Computed:            true,
			},
			"latency_endpoint": schema.StringAttribute{
				MarkdownDescription: "Endpoint used for latency measurement",
				Computed:            true,
			},
			"server_prefix": schema.StringAttribute{
				MarkdownDescription: "Prefix that can be used to for server names",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Datacenter ID",
			},
		},
	}
}

func (d *DataCenterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DataCenterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataCenterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	datacenterList, err := d.client.CloudClient().ListDatacenters(context.Background(), &cloudv1.ListDatacentersRequest{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list datacenters", err.Error())
		return
	}
	for _, datacenter := range datacenterList.Datacenters {
		if strings.EqualFold(datacenter.Short, data.Short.ValueString()) {
			data.Id = types.StringValue(datacenter.Id)
			data.Name = types.StringValue(datacenter.Name)
			data.Short = types.StringValue(datacenter.Short)
			data.RegionID = types.StringValue(datacenter.Region.Id)
			data.LatencyEndpoint = types.StringValue(datacenter.LatencyEndpoint)
			data.ServerPrefix = types.StringValue(datacenter.ServerPrefix)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}
	resp.Diagnostics.AddError("Unable to find datacenter", "Unable to find datacenter with short name: "+data.Short.ValueString())
}
