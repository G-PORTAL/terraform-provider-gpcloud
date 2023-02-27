package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	typev1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/type/v1"
	"context"
	"fmt"
	"github.com/G-PORTAL/gpcloud-go/pkg/gpcloud/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SSHKey{}
var _ resource.ResourceWithImportState = &SSHKey{}

func NewSSHKey() resource.Resource {
	return &SSHKey{}
}

// SSHKey defines the resource implementation.
type SSHKey struct {
	client *client.Client
}

// SSHKeyModel describes the resource data model.
type SSHKeyModel struct {
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"ssh_key_type"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`

	Id types.String `tfsdk:"id"`
}

func (r *SSHKey) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sshkey"
}

func (r *SSHKey) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "The SSH Key can be referenced in Node deployment to be used for SSH access to the node.\n\n" +
			"In case the SSH Key already exists remotely, use the `import` command provided by terraform cli to import the resource into the state file.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the SSH Key",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ssh_key_type": schema.StringAttribute{
				MarkdownDescription: "Type of the SSH Key",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "SSHKey Fingerprint",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "SSH Public Key",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SSHKey ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SSHKey) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSHKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *SSHKeyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := &cloudv1.CreateUserSSHKeyRequest{
		Name:      data.Name.ValueString(),
		PublicKey: data.PublicKey.ValueString(),
	}

	createResponse, err := r.client.CloudClient().CreateUserSSHKey(context.Background(), createRequest)
	if err != nil && strings.Contains(err.Error(), "AlreadyExists") {
		sshKeyResponse, err := r.client.CloudClient().ListUserSSHKeys(context.Background(), &cloudv1.ListUserSSHKeysRequest{})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list ssh keys, got error: %s", err))
			return
		}
		for _, sshKey := range sshKeyResponse.SshKeys {
			if sshKey.Name == data.Name.ValueString() {
				data.writeNewKey(sshKey)
				break
			}
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ssh key, got error: %s", err))
		return
	}
	data.writeNewKey(createResponse.SshKey)
	tflog.Trace(ctx, "SSHKey created")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SSHKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *SSHKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	sshKeyResponse, err := r.client.CloudClient().ListUserSSHKeys(context.Background(), &cloudv1.ListUserSSHKeysRequest{})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list ssh keys, got error: %s", err))
		return
	}
	for _, sshKey := range sshKeyResponse.SshKeys {
		if !data.Id.IsNull() {
			if data.Id.ValueString() == sshKey.Id {
				data.writeNewKey(sshKey)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
	}
}

func (r *SSHKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Client Error", "Unable to update ssh key")
}

func (r *SSHKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *SSHKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CloudClient().DeleteUserSSHKey(context.Background(), &cloudv1.DeleteUserSSHKeyRequest{
		Id: data.Id.ValueString(),
	})
	if err != nil && status.Code(err) == codes.NotFound {
		resp.Diagnostics.AddWarning("Client Warning", fmt.Sprintf("SSHKey that should be deleted does not exist: %s", err))
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *SSHKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (sshKeyModel *SSHKeyModel) writeNewKey(sshKey *typev1.SSHKey) {
	sshKeyModel.Id = types.StringValue(sshKey.Id)
	sshKeyModel.Name = types.StringValue(sshKey.Name)
	sshKeyModel.Type = types.StringValue(sshKey.Type.String())
	sshKeyModel.Fingerprint = types.StringValue(*sshKey.Fingerprint)
}
