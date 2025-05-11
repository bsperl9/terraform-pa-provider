package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: Provider,
	})
}

func Provider() *schema.Provider {
	prov := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"server": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   false,
			},
			"port": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1433",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pa_unit": resourceUnit(),
			"pa_department": resourceDepartment(),
		},
		ConfigureContextFunc: providerConfigure,
	}

	return prov
}


func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	connStr := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=SOADB",
		d.Get("server").(string),
		d.Get("username").(string),
		d.Get("password").(string),
		d.Get("port").(string))

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return db, nil
}
