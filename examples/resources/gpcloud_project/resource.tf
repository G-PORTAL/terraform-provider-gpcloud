resource "gpcloud_project" "example" {
  name        = "example-project"
  description = "Example project"
  environment = "PROJECT_ENVIRONMENT_DEVELOPMENT"
}