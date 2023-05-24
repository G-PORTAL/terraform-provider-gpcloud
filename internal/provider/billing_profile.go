package provider

import (
	cloudv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/cloud/v1"
	paymentv1 "buf.build/gen/go/gportal/gportal-cloud/protocolbuffers/go/gpcloud/api/payment/v1"
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
	"strings"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BillingProfile{}
var _ resource.ResourceWithImportState = &BillingProfile{}

func NewBillingProfile() resource.Resource {
	return &BillingProfile{}
}

// BillingProfile defines the resource implementation.
type BillingProfile struct {
	client *client.Client
}

// BillingProfileModel describes the resource data model.
type BillingProfileModel struct {
	Name         types.String `tfsdk:"name"`
	CountryCode  types.String `tfsdk:"country_code"`
	State        types.String `tfsdk:"state"`
	Street       types.String `tfsdk:"street"`
	City         types.String `tfsdk:"city"`
	Postcode     types.String `tfsdk:"postcode"`
	BillingEmail types.String `tfsdk:"billing_email"`
	CompanyName  types.String `tfsdk:"company_name"`
	CompanyVatId types.String `tfsdk:"company_vat_id"`
	Id           types.String `tfsdk:"id"`
}

func (r *BillingProfile) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_billing_profile"
}

func (r *BillingProfile) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "The datasource billing profile represents the Billing Profiles within GPCloud.\n\n" +
			"Billing Profiles are defined on user level and can be applied to different projects.\n" +
			"Each billing profile is receiving their own invoice at the end of the billing period.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Full name",
				Required:            true,
			},
			"country_code": schema.StringAttribute{
				MarkdownDescription: "Country Code",
				Required:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "State",
				Required:            true,
			},
			"street": schema.StringAttribute{
				MarkdownDescription: "Street",
				Required:            true,
			},
			"city": schema.StringAttribute{
				MarkdownDescription: "City",
				Required:            true,
			},
			"postcode": schema.StringAttribute{
				MarkdownDescription: "Postcode",
				Required:            true,
			},
			"billing_email": schema.StringAttribute{
				MarkdownDescription: "Email-Address to receive invoices",
				Optional:            true,
			},
			"company_name": schema.StringAttribute{
				MarkdownDescription: "Company Name",
				Optional:            true,
			},
			"company_vat_id": schema.StringAttribute{
				MarkdownDescription: "Vat ID",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Billing Profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BillingProfile) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BillingProfile) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *BillingProfileModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	company := data.CompanyName.ValueString()
	vatID := data.CompanyVatId.ValueString()

	createResponse, err := r.client.PaymentClient().CreateBillingProfile(context.Background(), &paymentv1.CreateBillingProfileRequest{
		Name:         data.Name.ValueString(),
		Company:      &company,
		VatId:        &vatID,
		CountryCode:  data.CountryCode.ValueString(),
		State:        data.State.ValueString(),
		Street:       data.Street.ValueString(),
		City:         data.City.ValueString(),
		Postcode:     data.Postcode.ValueString(),
		BillingEmail: data.BillingEmail.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create billing profile, got error: %s", err))
		return
	}
	data.write(createResponse.BillingProfile)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, fmt.Sprintf("Created Billing Profile: %s", data.Id.ValueString()))
}

func (r *BillingProfile) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *BillingProfileModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	billingProfileResponse, err := r.client.PaymentClient().ListBillingProfiles(context.Background(), &paymentv1.ListBillingProfilesRequest{})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get billing profile, got error: %s", err))
		return
	}
	for _, profile := range billingProfileResponse.BillingProfiles {
		if profile.Id == data.Id.ValueString() {
			data.write(profile)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}
	resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get billing profile, got error: %s", err))
}

func (r *BillingProfile) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *BillingProfileModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	company := data.CompanyName.ValueString()
	vatID := data.CompanyVatId.ValueString()
	updateResponse, err := r.client.PaymentClient().UpdateBillingProfile(context.Background(), &paymentv1.UpdateBillingProfileRequest{
		Name:         data.Name.ValueString(),
		Company:      &company,
		VatId:        &vatID,
		CountryCode:  data.CountryCode.ValueString(),
		State:        data.State.ValueString(),
		Street:       data.Street.ValueString(),
		City:         data.City.ValueString(),
		Postcode:     data.Postcode.ValueString(),
		BillingEmail: data.BillingEmail.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update billing profile, got error: %s", err))
		return
	}

	data.write(updateResponse.BillingProfile)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, fmt.Sprintf("Updated billing profile: %s", data.Id.ValueString()))
}

func (r *BillingProfile) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *BillingProfileModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.PaymentClient().DeleteBillingProfile(context.Background(), &paymentv1.DeleteBillingProfileRequest{
		Id: data.Id.ValueString(),
	})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		resp.Diagnostics.AddWarning("Client Warning", fmt.Sprintf("Billing Profile that should be deleted does not exist: %s", err))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete billing profile, got error: %s", err))
		return
	}
}

func (r *BillingProfile) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (data *BillingProfileModel) write(profile *cloudv1.BillingProfile) {
	data.Id = types.StringValue(profile.Id)
	data.Name = types.StringValue(profile.Name)
	data.CountryCode = types.StringValue(profile.CountryCode)
	data.State = types.StringValue(profile.State)
	data.Street = types.StringValue(profile.Street)
	data.City = types.StringValue(profile.City)
	data.Postcode = types.StringValue(profile.Postcode)
	data.BillingEmail = types.StringValue(profile.BillingEmail)
	if profile.Company != nil {
		data.CompanyName = types.StringValue(profile.Company.Name)
		if profile.Company.VatId != nil {
			data.CompanyVatId = types.StringValue(*profile.Company.VatId)
		}
	}
}
