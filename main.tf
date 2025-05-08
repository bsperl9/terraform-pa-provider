provider "pa" {
  server   = "172.29.176.1" # SQL Server hostname
  port     = "1433"         # Optional, defaults to 1433
  username = "pa-provider"
  password = "pa-provider-pass"
}

resource "pa_unit" "example" {
  pu_id       = 1
  description = "Example Unit"
}

resource "pa_unit" "example2" {
  pu_id       = 2
  description = "Scooby doo"
}