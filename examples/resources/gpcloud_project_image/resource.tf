resource "gpcloud_project_image" "example" {
  project_id = "d6e1052e-20e2-4c70-8bfd-4af4796805d3"
  name       = "Ubuntu 22.04"
  source     = "./jammy-server-cloudimg-amd64.tar.gz"
}