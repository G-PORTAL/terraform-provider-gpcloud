resource "gpcloud_billing_profile" "example" {
  name           = "John Smith"
  street         = "132, My Street"
  city           = "Kingston"
  postcode       = "12401"
  state          = "New York"
  country_code   = "US"
  billing_email  = "invoices@example.com"
  company_name   = "Example Inc."
  company_vat_id = "US123456789"
}