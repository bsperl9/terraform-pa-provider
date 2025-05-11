terraform {
  required_providers {
    pa = {
      source = "local/bts/pa"
    }
  }
}

provider "pa" {
  server   = "172.29.176.1" # SQL Server hostname
  port     = "1433"         # Optional, defaults to 1433
  username = "pa-provider"
  password = "pa-provider-pass"
}