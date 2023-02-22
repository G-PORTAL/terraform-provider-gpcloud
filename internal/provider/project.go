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
	"strings"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Project{}
var _ resource.ResourceWithImportState = &Project{}

func NewProject() resource.Resource {
	return &Project{}
}

// Project defines the resource implementation.
type Project struct {
	client *client.Client
}

// ProjectModel describes the resource data model.
type ProjectModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Environment types.String `tfsdk:"environment"`
	Id          types.String `tfsdk:"id"`
}

func (r *Project) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *Project) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Project",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Project Description",
				Required:            true,
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Project Environment",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.ProjectEnvironmentValidator{},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *Project) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *Project) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createResponse, err := r.client.CloudClient().CreateProject(context.Background(), &cloudv1.CreateProjectRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Environment: cloudv1.ProjectEnvironment(cloudv1.ProjectEnvironment_value[data.Environment.ValueString()]),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}
	data.write(createResponse.Project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, fmt.Sprintf("Created project: %s", data.Id.ValueString()))
}

func (r *Project) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectResponse, err := r.client.CloudClient().GetProject(context.Background(), &cloudv1.GetProjectRequest{
		Id: data.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get project, got error: %s", err))
		return
	}
	data.write(projectResponse.Project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Project) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateResponse, err := r.client.CloudClient().UpdateProject(context.Background(), &cloudv1.UpdateProjectRequest{
		Id:          data.Id.ValueString(),
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Environment: cloudv1.ProjectEnvironment(cloudv1.ProjectEnvironment_value[data.Environment.ValueString()]),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	data.write(updateResponse.Project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, fmt.Sprintf("Updated project: %s", data.Id.ValueString()))
}

func (r *Project) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CloudClient().DeleteProject(context.Background(), &cloudv1.DeleteProjectRequest{
		Id: data.Id.ValueString(),
	})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		resp.Diagnostics.AddWarning("Client Warning", fmt.Sprintf("Project that should be deleted does not exist: %s", err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *Project) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (data *ProjectModel) write(project *cloudv1.Project) {
	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Environment = types.StringValue(project.Environment.String())
	if project.Description != "" {
		data.Description = types.StringValue(project.Description)
	}
}
