package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	"context"
	"fmt"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"github.com/G-PORTAL/terraform-provider-gpcloud/internal/gpcloudvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Node{}
var _ resource.ResourceWithImportState = &Node{}

func NewNode() resource.Resource {
	return &Node{}
}

// Node defines the resource implementation.
type Node struct {
	client *client.Client
}

// NodeModel describes the resource data model.
type NodeModel struct {
	ProjectID     types.String `tfsdk:"project_id"`
	FlavorID      types.String `tfsdk:"flavor_id"`
	DatacenterID  types.String `tfsdk:"datacenter_id"`
	Password      types.String `tfsdk:"password"`
	SSHKeyIDs     types.List   `tfsdk:"ssh_key_ids"`
	UserData      types.String `tfsdk:"user_data"`
	FQDN          types.String `tfsdk:"fqdn"`
	BillingPeriod types.String `tfsdk:"billing_period"`
	ImageID       types.String `tfsdk:"image_id"`
	IP            types.String `tfsdk:"ip"`
	Tags          types.Map    `tfsdk:"tags"`
	Id            types.String `tfsdk:"id"`
}

func (r *Node) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

func (r *Node) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Node is the representation of the Bare Metal Node that got created in the G-PORTAL Cloud.\n\n" +
			"Changing the Nodes Image ID will cause the Node to be destroyed and recreated.\n\n",

		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Node FQDN",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "Node Description",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"datacenter_id": schema.StringAttribute{
				MarkdownDescription: "Datacenter ID the node is located in",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password used for authentication",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_ids": schema.ListAttribute{
				MarkdownDescription: "SSH Keys used for authentication",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					gpcloudvalidator.UUIDListValidator{},
				},
			},
			"user_data": schema.StringAttribute{
				MarkdownDescription: "User Data to be provided for cloud-init",
				Optional:            true,
			},
			"fqdn": schema.StringAttribute{
				MarkdownDescription: "Fully Qualified Domain Name of the node",
				Required:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "IP Address of the node",
				Computed:            true,
			},
			"billing_period": schema.StringAttribute{
				MarkdownDescription: "Billing Configuration",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					gpcloudvalidator.BillingPeriodValidator{},
				},
			},
			"image_id": schema.StringAttribute{
				MarkdownDescription: "Image ID to install the node with",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Node Tags",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Node ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *Node) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *Node) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *NodeModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	createRequest := &cloudv1.CreateNodeRequest{
		Fqdns:         []string{data.FQDN.ValueString()},
		ProjectId:     data.ProjectID.ValueString(),
		FlavourId:     data.FlavorID.ValueString(),
		DatacenterId:  data.DatacenterID.ValueString(),
		ImageId:       data.ImageID.ValueString(),
		BillingPeriod: cloudv1.BillingPeriod(cloudv1.BillingPeriod_value[data.BillingPeriod.ValueString()]),
	}

	if !data.Password.IsNull() {
		passwd := data.Password.ValueString()
		createRequest.Password = &passwd
	}
	if !data.SSHKeyIDs.IsNull() {
		for _, sshKeyID := range data.SSHKeyIDs.Elements() {
			if sshKeyIDString, ok := sshKeyID.(types.String); ok {
				createRequest.SshKeyIds = append(createRequest.SshKeyIds, sshKeyIDString.ValueString())
			}
		}
	}
	if !data.UserData.IsNull() {
		userData := data.UserData.ValueString()
		createRequest.UserData = &userData
	}

	createResponse, err := r.client.CloudClient().CreateNode(context.Background(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create node, got error: %s", err))
		return
	}
	nodeData := createResponse.Nodes[0]
	data.write(nodeData)

	nodeIP := data.getPrimaryIP(nodeData)
	timeoutAfter := time.Now().Add(5 * time.Minute)
	for nodeIP == nil {
		if time.Now().After(timeoutAfter) {
			resp.Diagnostics.AddError("Timeout Error", "Unable to get node IP address")
			return
		}
		time.Sleep(time.Second * 10)
		if getNodeResponse, err := r.client.CloudClient().GetNode(context.Background(), &cloudv1.GetNodeRequest{
			Id:        data.Id.ValueString(),
			ProjectId: data.ProjectID.ValueString(),
		}); err == nil {
			data.write(getNodeResponse.Node)
			nodeIP = data.getPrimaryIP(getNodeResponse.Node)
		}
	}

	// If tags should be added, update the node
	if len(data.Tags.Elements()) > 0 {
		updateRequest := &cloudv1.UpdateNodeRequest{
			Id:        nodeData.Id,
			ProjectId: nodeData.ProjectId,
			Fqdn:      &nodeData.Fqdn,
			Tags:      map[string]string{},
		}
		for s, value := range data.Tags.Elements() {
			if stringValue, ok := value.(types.String); ok {
				updateRequest.Tags[s] = stringValue.ValueString()
			}
		}
		updateResponse, err := r.client.CloudClient().UpdateNode(context.Background(), updateRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update node after creation, got error: %s", err))
			return
		}
		data.write(updateResponse.Node)
	}

	tflog.Trace(ctx, fmt.Sprintf("Created node with ID: %s", data.Id.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Node) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *NodeModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	if !data.Id.IsNull() {
		nodeResponse, err := r.client.CloudClient().GetNode(context.Background(), &cloudv1.GetNodeRequest{
			Id:        data.Id.ValueString(),
			ProjectId: data.ProjectID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get node, got error: %s", err))
			return
		}
		data.write(nodeResponse.Node)

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}
}

func (r *Node) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *NodeModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	fqdn := data.FQDN.ValueString()
	updateRequest := &cloudv1.UpdateNodeRequest{
		Id:        data.Id.ValueString(),
		ProjectId: data.ProjectID.ValueString(),
		Fqdn:      &fqdn,
		Tags:      map[string]string{},
	}

	for s, value := range data.Tags.Elements() {
		if stringValue, ok := value.(types.String); ok {
			updateRequest.Tags[s] = stringValue.ValueString()
		}
	}

	updateResponse, err := r.client.CloudClient().UpdateNode(context.Background(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}
	data.write(updateResponse.Node)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, fmt.Sprintf("Updated node: %s", data.Id.ValueString()))
}

func (r *Node) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *NodeModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CloudClient().DestroyNode(context.Background(), &cloudv1.DestroyNodeRequest{
		Id:        data.Id.ValueString(),
		ProjectId: data.ProjectID.ValueString(),
	})
	if err != nil && status.Code(err) != codes.NotFound {
		resp.Diagnostics.AddWarning("Client Warning", fmt.Sprintf("Node that should be deleted does not exist: %s", err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *Node) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (nodeModel *NodeModel) getPrimaryIP(node *cloudv1.Node) *string {
	for _, networkInterface := range node.NetworkInterfaces {
		for _, address := range networkInterface.IpAddresses {
			ip := address
			return &ip
		}
	}
	return nil
}

func (nodeModel *NodeModel) write(node *cloudv1.Node) {
	nodeModel.ProjectID = types.StringValue(node.ProjectId)
	nodeModel.FlavorID = types.StringValue(node.Flavour.Id)
	nodeModel.DatacenterID = types.StringValue(node.Datacenter.Id)
	nodeModel.FQDN = types.StringValue(node.Fqdn)
	nodeModel.BillingPeriod = types.StringValue(node.BillingPeriod.String())
	nodeModel.ImageID = types.StringValue(node.Image.Id)
	nodeModel.Id = types.StringValue(node.Id)
	if nodeIP := nodeModel.getPrimaryIP(node); nodeIP != nil {
		nodeModel.IP = types.StringValue(*nodeIP)
	}
}
