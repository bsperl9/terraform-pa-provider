package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	_ "github.com/microsoft/go-mssqldb"
)

type Department struct {
	Dept_Id       	int64
	Description  	string
	ExtendedInfo 	string
	TimeZone 		string
	Tag				string
}

var (
	cachedDepartments map[int64]*Department
)


const (
	queryLoadDepartments = `
		SELECT Dept_Id, Dept_Desc, Extended_Info, Time_Zone, Tag 
		FROM dbo.Departments 
		WHERE Dept_Id >= 0 
		ORDER BY Dept_Id DESC;`

	queryCreateDepartment = `
		EXEC @return_value = [dbo].[spEM_CreateDepartment]
		    @Description = @param_desc,
		    @User_Id = @param_user_id,
		    @Dept_Id = @out_deptId OUTPUT;

		IF @return_value != 0 OR @out_deptId IS NULL
		BEGIN
		    RETURN;
		END

		UPDATE dbo.Departments_Base
		SET Dept_Desc = ISNULL(@param_desc, Dept_Desc),
		    Extended_Info = ISNULL(@param_ext_info, Extended_Info),
		    Time_Zone = ISNULL(@param_tz, Time_Zone),
		    Tag = ISNULL(@param_tag, Tag)
		WHERE Dept_Id = @out_deptId;
		`

	queryUpdateDepartment = `
		UPDATE dbo.Departments_Base SET
			Dept_Desc = ISNULL(@param_desc, Dept_Desc),
			Extended_Info = ISNULL(@param_ext_info, Extended_Info),
			Time_Zone = ISNULL(@param_tz, Time_Zone),
			Tag = ISNULL(@param_tag, Tag)
		WHERE Dept_Id = @param_deptId`

	queryDeleteDepartment = "DELETE FROM SOADB.dbo.Departments_Base WHERE Dept_Id = @param_deptId"
)

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
            "dept_id": {
                Type:     schema.TypeInt,
                Computed: true, // Not settable by user
            },
            "description": {
                Type:     schema.TypeString,
                Required: true,
                ValidateFunc: validateTitle(),
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
            "tag": {
                Type:     schema.TypeString,
                Optional: true,
            },
        },
	}
}

func loadDepartmentsCache(ctx context.Context, m interface{}) error {
	var err error
	db := getDB(m)
	loadOnce.Do(func() {
		cachedDepartments = make(map[int64]*Department)

		rows, queryErr := db.QueryContext(ctx, queryLoadDepartments)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var dept Department
			var description, extendedInfo, timeZone, tag sql.NullString

			if scanErr := rows.Scan(&dept.Dept_Id, &description, &extendedInfo, &timeZone, &tag); scanErr != nil {
				err = scanErr
				return
			}
			dept.Description = nullableStringToString(description)
			dept.ExtendedInfo = nullableStringToString(extendedInfo)
			dept.TimeZone = nullableStringToString(timeZone)
			dept.Tag = nullableStringToString(tag)
			cachedDepartments[dept.Dept_Id] = &dept
		}
	})
	return err
}

func resourceDepartmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)

	if err := loadDepartmentsCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}

	var description sql.NullString
	var extendedInfo sql.NullString
	var timeZone sql.NullString
	var tag sql.NullString
	var deptID int64
	var returnValue int64

	description = stringToNullString(d.Get("description").(string))
	extendedInfo = stringToNullString(d.Get("extended_info").(string))
	timeZone = stringToNullString(d.Get("time_zone").(string))
	tag = stringToNullString(d.Get("tag").(string))
	userId := 1	

	_, err := db.ExecContext(ctx, queryCreateDepartment,
		sql.Named("return_value", sql.Out{Dest: &returnValue}),
		sql.Named("param_desc", description),
		sql.Named("param_user_id", userId),
		sql.Named("out_deptId", sql.Out{Dest: &deptID}),
		sql.Named("param_ext_info", extendedInfo),
		sql.Named("param_tz", timeZone),
		sql.Named("param_tag", tag),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute create logic: %w", err))
	}
	if returnValue != 0 || deptID == 0 {
		return diag.FromErr(fmt.Errorf("stored procedure returned failure status: %d or null ID", returnValue))
	}

	cachedDepartments[deptID] = &Department{
		Dept_Id:      deptID,
		Description:  nullableStringToString(description),
		ExtendedInfo: nullableStringToString(extendedInfo),
		TimeZone:     nullableStringToString(timeZone),
		Tag:          nullableStringToString(tag),
	}

	d.Set("dept_id", int(deptID))
	d.SetId(int64ToString(deptID))
	return resourceDepartmentRead(ctx, d, m)
}

func resourceDepartmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := loadDepartmentsCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}
	id := int64(d.Get("dept_id").(int))

	dept, ok := cachedDepartments[id]
	if !ok {
		d.SetId("")
		return nil
	}

	d.Set("dept_id", id)
	d.Set("description", dept.Description)
	d.Set("extended_info", dept.ExtendedInfo)
	d.Set("time_zone", dept.TimeZone)
	d.Set("tag", dept.Tag)
	return nil
}

func resourceDepartmentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)

	id := int64(d.Get("dept_id").(int))
	description := d.Get("description").(string)
	extendedInfo := d.Get("extended_info").(string)
	timeZone := d.Get("time_zone").(string)
	tag := d.Get("tag").(string)
	userId := 1

	_, err := db.ExecContext(ctx, queryUpdateDepartment,
		sql.Named("param_deptId", id),
		sql.Named("param_desc", stringToNullString(description)),
		sql.Named("param_user_id", userId),
		sql.Named("param_ext_info", stringToNullString(extendedInfo)),
		sql.Named("param_tz", stringToNullString(timeZone)),
		sql.Named("param_tag", stringToNullString(tag)),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDepartmentRead(ctx, d, m)
}

func resourceDepartmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)

	id := int64(d.Get("dept_id").(int))

	_, err := db.ExecContext(ctx, queryDeleteDepartment, sql.Named("param_deptId", id))
	if err != nil {
		return diag.FromErr(err)
	}

	delete(cachedDepartments, id)
	d.SetId("")
	return nil
}