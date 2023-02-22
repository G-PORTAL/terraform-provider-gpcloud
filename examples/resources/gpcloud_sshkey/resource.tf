resource "gpcloud_sshkey" "example" {
  name       = "my-custom-ssh-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC9S52MGGlI+JyJ+2szNXJA70j9C1O1vuICAW5RZRdLr[...]"
}