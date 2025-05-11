package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"regexp"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	_ "github.com/microsoft/go-mssqldb"
)

var (
	cachedDepartments map[int]map[string]string
	loadOnce          sync.Once
)

const (
	queryLoadDepartments = `
		SELECT Dept_Id, Dept_Desc, Extended_Info, Time_Zone 
		FROM dbo.Departments 
		WHERE Dept_Id >= 0 
		ORDER BY Dept_Id DESC;`

	queryCreateDepartment = `
		EXEC @return_value = [dbo].[spEM_CreateDepartment]
		    @Description = @desc,
		    @User_Id = @userId,
		    @Dept_Id = @deptId OUTPUT;

		IF @return_value != 0 OR @deptId IS NULL
		BEGIN
		    RETURN;
		END

		UPDATE dbo.Departments_Base
		SET Dept_Desc = ISNULL(@desc, Dept_Desc),
		    Extended_Info = ISNULL(@extInfo, Extended_Info),
		    Time_Zone = ISNULL(@tz, Time_Zone)
		WHERE Dept_Id = @deptId;

		SELECT @deptId AS id;
		`

	queryUpdateDepartment = `
		UPDATE dbo.Departments_Base SET
			Dept_Desc = ISNULL(@desc, Dept_Desc),
			Extended_Info = ISNULL(@extInfo, Extended_Info),
			Time_Zone = ISNULL(@tz, Time_Zone)
		WHERE Dept_Id = @deptId`

	queryDeleteDepartment = "DELETE FROM SOADB.dbo.Departments_Base WHERE Dept_Id = @p1"
)

func validateDescription() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`^[\w\-\(\)]+( [\w\-\(\)]+)*$`),
		"description can only contain alphanumeric characters, spaces, dashes (-), underscores (_), and parentheses () and must not start or end with spaces",
	)
}

func validateTimeZone() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"Eastern Standard Time", "Pacific Standard Time", "UTC",
	}, false)
}

func resourceDepartment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDepartmentCreate,
		ReadContext:   resourceDepartmentRead,
		UpdateContext: resourceDepartmentUpdate,
		DeleteContext: resourceDepartmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
            "pu_id": {
                Type:     schema.TypeInt,
                Computed: true, // Not settable by user
            },
            "description": {
                Type:     schema.TypeString,
                Required: true,
                ValidateFunc: validateDescription(),
            },
            "extended_info": {
                Type:     schema.TypeString,
                Optional: true,
            },
            "time_zone": {
                Type:     schema.TypeString,
                Optional: true,
                ValidateFunc: validateTimeZone(),
            },
        },
	}
}

func getDB(m interface{}) *sql.DB {
	return m.(*sql.DB)
}

func loadDepartmentsCache(ctx context.Context, db *sql.DB) error {
	var err error
	loadOnce.Do(func() {
		cachedDepartments = make(map[int]map[string]string)

		rows, queryErr := db.QueryContext(ctx, queryLoadDepartments)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var desc, info, tz sql.NullString
			if scanErr := rows.Scan(&id, &desc, &info, &tz); scanErr != nil {
				err = scanErr
				return
			}
			cachedDepartments[id] = map[string]string{
				"description":   desc.String,
				"extended_info": info.String,
				"time_zone":     tz.String,
			}
		}
	})
	return err
}

func resourceDepartmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	description := d.Get("description").(string)
	extendedInfo := d.Get("extended_info").(string)
	timeZone := d.Get("time_zone").(string)
	userId := 1

	var deptID int
	var returnValue int

	_, err := db.ExecContext(ctx, queryCreateDepartment,
		sql.Named("return_value", sql.Out{Dest: &returnValue}),
		sql.Named("desc", description),
		sql.Named("userId", userId),
		sql.Named("deptId", sql.Out{Dest: &deptID}),
		sql.Named("extInfo", extendedInfo),
		sql.Named("tz", timeZone),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute create logic: %w", err))
	}
	if returnValue != 0 || deptID == 0 {
		return diag.FromErr(fmt.Errorf("stored procedure returned failure status: %d or null ID", returnValue))
	}

	d.SetId(strconv.Itoa(deptID))
	return resourceDepartmentRead(ctx, d, m)
}

func resourceDepartmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	id, _ := strconv.Atoi(d.Id())

	if err := loadDepartmentsCache(ctx, db); err != nil {
		return diag.FromErr(err)
	}

	dept, ok := cachedDepartments[id]
	if !ok {
		d.SetId("")
		return nil
	}

	d.Set("id", id)
	d.Set("description", dept["description"])
	d.Set("extended_info", dept["extended_info"])
	d.Set("time_zone", dept["time_zone"])
	return nil
}

func resourceDepartmentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	id, _ := strconv.Atoi(d.Id())
	description := d.Get("description").(string)
	extendedInfo := d.Get("extended_info").(string)
	timeZone := d.Get("time_zone").(string)
	userId := 1

	_, err := db.ExecContext(ctx, queryUpdateDepartment,
		sql.Named("deptId", id),
		sql.Named("desc", description),
		sql.Named("userId", userId),
		sql.Named("extInfo", extendedInfo),
		sql.Named("tz", timeZone),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDepartmentRead(ctx, d, m)
}

func resourceDepartmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	id, _ := strconv.Atoi(d.Id())

	_, err := db.ExecContext(ctx, queryDeleteDepartment, id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}