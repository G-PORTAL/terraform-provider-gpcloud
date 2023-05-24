resource "gpcloud_project" "example" {
  name               = "example-project"
  description        = "Example project"
  environment        = "PROJECT_ENVIRONMENT_DEVELOPMENT"
  billing_profile_id = "40c5b014-7817-4e0f-8957-ee0551b5c07f"
}