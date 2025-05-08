# Terraform PA Provider

This is a custom Terraform provider for managing Units in a SQL Server database.

## Requirements

- Go 1.21 or higher
- Terraform 0.12 or higher
- SQL Server instance

## Building the Provider

1. Clone the repository
2. Run `go mod tidy` to download dependencies
3. Run `go build -o terraform-provider-pa` to build the provider

## Using the Provider

```hcl
terraform {
  required_providers {
    pa = {
      source = "local/pa"
      version = "1.0"
    }
  }
}

provider "pa" {
  server   = "localhost"     # SQL Server hostname
  port     = "1433"         # Optional, defaults to 1433
  username = "your_username"
  password = "your_password"
}

resource "pa_unit" "example" {
  pu_id       = 1
  description = "Example Unit"
}
```

## Importing Existing Resources

To import an existing unit:

```bash
terraform import pa_unit.example 1  # Where 1 is the PU_Id
```
