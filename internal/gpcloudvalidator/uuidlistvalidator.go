package gpcloudvalidator

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type UUIDListValidator struct {
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v UUIDListValidator) Description(ctx context.Context) string {
	return "Has to be a list of valid UUIDv4s"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v UUIDListValidator) MarkdownDescription(ctx context.Context) string {
	return "Has to be a list of valid UUIDv4s"
}

// ValidateList runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v UUIDListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	for _, value := range req.ConfigValue.Elements() {
		if value.IsUnknown() || value.IsNull() {
			continue
		}
		if _, err := uuid.Parse(value.String()); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid UUID",
				fmt.Sprintf("The value %q is not a valid UUIDv4.", value.String()),
			)
		}
	}
}
