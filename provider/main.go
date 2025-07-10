package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"sync"
	"strconv"
)

var (
	loadOnce          sync.Once
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
			"database": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "SOADB",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pa_test_unit": resourceTestUnit(),
			"pa_department": resourceDepartment(),
			"pa_line": resourceLine(),
		},
		ConfigureContextFunc: providerConfigure,
	}

	return prov
}


func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	connStr := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s",
		d.Get("server").(string),
		d.Get("username").(string),
		d.Get("password").(string),
		d.Get("port").(string),
		d.Get("database").(string))

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return db, nil
}

func getDB(m interface{}) *sql.DB {
	return m.(*sql.DB)
}

func stringToNullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableInt64ToInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return -1
	}
	return value.Int64
}

func int64ToNullInt64(value int64) sql.NullInt64 {
	if value == -1 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func stringToInt64(value string) (int64, error) {	
	return strconv.ParseInt(value, 10, 64)
}


//
//func nullableToInterface(value interface{}) interface{} {
//	switch v := value.(type) {
//	case sql.NullString:
//		if v.Valid {
//			return v.String
//		}
//	case sql.NullInt64:
//		if v.Valid {
//			return v.Int64
//		}
//	case sql.NullBool:
//		if v.Valid {
//			return v.Bool
//		}
//	case sql.NullFloat64:
//		if v.Valid {
//			return v.Float64
//		}
//	}
//	return nil
//}
//