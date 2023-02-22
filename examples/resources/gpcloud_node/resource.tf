resource "gpcloud_node" "example" {
  project_id     = "aad60ae1-f27f-4f46-9d53-a87e230d4c28"
  fqdn           = "my-node.example.com"
  image_id       = "8e41255d-2ee6-4258-b658-ce3558911216"
  ssh_key_ids    = ["90b5d5f1-fc37-457d-9060-a94349be5b5d"]
  flavor_id      = "1ec0e53e-c3c3-4a5e-af67-4d2d138cb042"
  datacenter_id  = "ea616457-d94c-4f44-a99f-3226310e7d23"
  billing_period = "BILLING_PERIOD_MONTHLY"
  tags = {
    "my-custom-tag" = "some-value"
  }
}