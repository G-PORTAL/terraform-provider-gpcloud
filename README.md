# Terraform Provider GPCloud

This repository is a Terraform provider for [GPCloud](https://g-portal.cloud).
It is intended to allow an Infrastructure-as-Code definition of the GPCloud services.

Implemented Resources:
- [x] `gpcloud_node` - The GPCloud Node resource
- [x] `gpcloud_project` - The GPCloud Project resource
- [x] `gpcloud_project_image` - The GPCloud Project Image resource (Custom image)
- [x] `gpcloud_sshkey` - The GPCloud SSH-Key resource

Implemented Data sources:
- [x] `gpcloud_project` - The GPCloud Project data source (read-only)
- [x] `gpcloud_flavour` - The GPCloud Flavour data source
- [x] `gpcloud_datacenter` - The GPCloud Datacenter data source
- [x] `gpcloud_image` - The GPCloud Image data source (Official images)


## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.18

## Building The GPCloud Provider manually

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Using the provider

This provider is published on the [Terraform Registry](https://registry.terraform.io/providers/g-portal/gpcloud/latest).

To use the GPCloud Terraform Provider, all you need to do is to reference the published provider inside your terraform configuration.

```hcl
terraform {
  required_providers {
    gpcloud = {
      source = "G-PORTAL/gpcloud"
      # Ensure to use the latest version of the provider
      version = "0.1.2"
    }
  }
}
```

The full documentation for the provider can be found [here](https://registry.terraform.io/providers/g-portal/gpcloud/latest/docs) or inside the `docs/` directory.

## Developing the Provider

If you wish to contribute to the GPCloud Terraform Provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To reference the custom version of the provider built instead of the hosted one, setting `dev_override` inside the `$HOME/.terraformrc` file becomes useful:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/g-portal/gpcloud" = "~/go/bin"
  }
  direct {}
}
```

To generate or update documentation, run `go generate`.
