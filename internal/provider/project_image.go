package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	"context"
	"errors"
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
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectImage{}
var _ resource.ResourceWithImportState = &ProjectImage{}

var uploadClient = &http.Client{
	Transport: &http.Transport{
		DisableCompression:  true,
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 300 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

func NewProjectImage() resource.Resource {
	return &ProjectImage{}
}

// ProjectImage defines the resource implementation.
type ProjectImage struct {
	client *client.Client
}

// ProjectImageModel describes the resource data model.
type ProjectImageModel struct {
	Name      types.String `tfsdk:"name"`
	Source    types.String `tfsdk:"source"`
	ProjectID types.String `tfsdk:"project_id"`
	//AuthenticationTypes        types.String `tfsdk:"ssh_key_type"`

	Id types.String `tfsdk:"id"`
}

func (r *ProjectImage) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_image"
}

func (r *ProjectImage) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "ProjectImage",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name to be used for the ProjectImage",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Image location (either http or local path)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID to place the Image in",
				Required:            true,
				Validators: []validator.String{
					gpcloudvalidator.UUIDStringValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ProjectImage ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectImage) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectImage) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectImageModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	createRequest := &cloudv1.CreateProjectImageRequest{
		Id:   data.ProjectID.ValueString(),
		Name: data.Name.ValueString(),
		AuthenticationTypes: []cloudv1.AuthenticationType{
			cloudv1.AuthenticationType_AUTHENTICATION_TYPE_SSH,
		},
	}

	createResponse, err := r.client.CloudClient().CreateProjectImage(context.Background(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project image, got error: %s", err))
		return
	}

	imageSource, err := r.getSourceReader(data.Source.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to open image source: %s", err))
		return
	}
	defer imageSource.Close()

	if err := r.uploadNewImageSource(
		createResponse.Image.ImageUpload.UploadUrl,
		createResponse.Image.ImageUpload.Token,
		imageSource); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upload image: %s", err))
	}
	data.writeNewData(createResponse.Image)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectImage) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectImageModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectProjectImageResponse, err := r.client.CloudClient().ListProjectImages(context.Background(), &cloudv1.ListProjectImagesRequest{
		Id: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list project images, got error: %s", err))
		return
	}

	for _, image := range projectProjectImageResponse.Images {
		if image.Id == data.Id.ValueString() {
			data.Name = types.StringValue(image.Name)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}
}

func (r *ProjectImage) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectImageModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	//TODO: Handle project image updates
	resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update image"))
}

func (r *ProjectImage) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectImageModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.ProjectID.IsNull() {
		return
	}

	_, err := r.client.CloudClient().DeleteProjectImage(context.Background(), &cloudv1.DeleteProjectImageRequest{
		Id:        data.Id.ValueString(),
		ProjectId: data.ProjectID.ValueString(),
	})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		resp.Diagnostics.AddWarning("Client Warning", fmt.Sprintf("ProjectImage that should be deleted does not exist: %s", err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete image, got error: %s", err))
		return
	}
}

func (r *ProjectImage) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ProjectImage) getSourceReader(source string) (io.ReadCloser, error) {
	if strings.HasPrefix(source, "http") {
		imageSource, err := http.Get(source)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to download image from source: %s", err))
		}
		return imageSource.Body, nil
	}
	imageSource, err := os.Open(source)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to open image source: %s", err))
	}
	return imageSource, nil
}

func (r *ProjectImage) uploadNewImageSource(uploadURL, uploadToken string, imageSource io.ReadCloser) error {
	uploadRequest, err := http.NewRequest("POST", uploadURL, imageSource)
	if err != nil {
		return err
	}
	uploadRequest.Header.Set("Content-Type", "application/octet-stream")
	uploadRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", uploadToken))
	if _, err := uploadClient.Do(uploadRequest); err != nil {
		return err
	}
	return nil
}

func (imageModel *ProjectImageModel) writeNewData(image *cloudv1.Image) {
	imageModel.Id = types.StringValue(image.Id)
	imageModel.Name = types.StringValue(image.Name)
	if image.Project != nil {
		imageModel.ProjectID = types.StringValue(image.Project.Id)
	}
}
