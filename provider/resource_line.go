package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	_ "github.com/microsoft/go-mssqldb"
)

type Line struct {
	Line_Id       		int64
	Description  		string
	ExtendedInfo 		string
	ExternalLink 		string
	SecurityGroup_Id	int64
	SecurityGroup		string
	Dept_Id				int64
	Department   		string
}

var (
	cachedLines map[int64]*Line
)

const (
	queryLoadLines = `
		SELECT	PLB.PL_Id, PLB.PL_Desc, PLB.Extended_Info, PLB.External_Link, PLB.Group_Id, SG.Group_Desc,
				DB.Dept_Id, DB.Dept_Desc
		FROM dbo.Prod_Lines_Base AS PLB
		JOIN dbo.Departments_Base AS DB ON DB.Dept_Id = PLB.Dept_Id
		LEFT JOIN dbo.Security_Groups AS SG ON SG.Group_Id = PLB.Group_Id
		WHERE PL_Id >= 0
		AND PL_Desc !='<PL Deleted>'
		ORDER BY PL_Id DESC;
		`

	queryCreateLine = `
		--DECLARE	@return_value		int,
		--		@out_PL_Id			int;
		
		--DECLARE @param_dept_id		int	= 25,
		--		@param_pl_desc		NVARCHAR(255)	= 'Line1',
		--		@param_ext_link		NVARCHAR(255)	= 'https://www.google.com',
		--		@param_ext_info		NVARCHAR(255)	= 'some extended info',
		--		@param_group_id		NVARCHAR(255)	= NULL,
		--		@param_user_id		int				= 1;
		
		DECLARE @dept_desc			NVARCHAR(255),
				@sg_desc			NVARCHAR(255);
		
		SET @dept_desc = (
			SELECT DB.Dept_Desc FROM dbo.Departments_Base AS DB WHERE DB.Dept_Id = @param_dept_id);
		
		SET @sg_desc = (
			SELECT SG.Group_Desc FROM dbo.Security_Groups AS SG WHERE SG.Group_Id = @param_group_id);
		
		
		EXEC	@return_value = [dbo].[spLocal_Provider_CreateLine]
				@Dept_Desc = @dept_desc,
				@PL_Desc = @param_pl_desc,
				@External_Link = @param_ext_link,
				@Extended_Info = @param_ext_info,
				@Group_Desc = @sg_desc,
				@User_Id = @param_user_id,
				@PL_Id = @out_PL_Id OUTPUT
		`
	queryUpdateLine = queryCreateLine

	queryDeleteLine = `
		EXEC	@return_value = [dbo].[spEM_DropLine]
				@PL_Id = @param_pl_id,
				@User_Id = @param_user_id;
	`

	queryGetDeptId = `
		SELECT Dept_Id FROM dbo.Departments_Base AS DB
		WHERE DB.Dept_Desc = @param_dept_desc;
	`

	queryGetLineId = `
		SELECT PL_Id FROM dbo.Prod_Lines_Base AS PLB
		WHERE PLB.PL_Desc = @param_pl_desc;
	`
)

func resourceLine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceLineCreate,
		ReadContext:   resourceLineRead,
		UpdateContext: resourceLineUpdate,
		DeleteContext: resourceLineDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
            "line_id": {
                Type:     schema.TypeInt,
                Computed: true, // Not settable by user
            },
            "description": {
                Type:     schema.TypeString,
                Required: true,
                ValidateFunc: validateTitle(),
            },
			"department_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
            "extended_info": {
                Type:     schema.TypeString,
                Optional: true,
				ValidateFunc: validateVarchar255(),
            },
            "external_link": {
                Type:     schema.TypeString,
                Optional: true,
                ValidateFunc: validateVarchar255(),
            },
			"security_group_id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
        },
	}
}

func loadLinesCache(ctx context.Context, m interface{}) error {
	var err error
	db := getDB(m)
	loadOnce.Do(func() {
		cachedLines = make(map[int64]*Line)

		rows, queryErr := db.QueryContext(ctx, queryLoadLines)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var line Line
			var description, extendedInfo, externalLink, securityGroup, department sql.NullString
			var securityGroupID sql.NullInt64
			var deptID sql.NullInt64
			if scanErr := rows.Scan(&line.Line_Id, &description, &extendedInfo, &externalLink, &securityGroupID, &securityGroup, &deptID, &department); scanErr != nil {
				err = scanErr
				return
			}
			line.Description = nullableStringToString(description)
			line.ExtendedInfo = nullableStringToString(extendedInfo)
			line.ExternalLink = nullableStringToString(externalLink)
			line.SecurityGroup = nullableStringToString(securityGroup)
			line.SecurityGroup_Id = nullableInt64ToInt64(securityGroupID)
			line.Dept_Id = nullableInt64ToInt64(deptID)
			line.Department = nullableStringToString(department)
			cachedLines[line.Line_Id] = &line
		}
	})
	return err
}

func resourceLineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	if err := loadLinesCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}
	description := d.Get("description").(string)
	dept_id := int64(d.Get("department_id").(int))
	extendedInfo := d.Get("extended_info").(string)
	sg_id := int64(d.Get("security_group_id").(int))
	externalLink := d.Get("external_link").(string)
	userId := 1

	var returnValue sql.NullInt64
	var outPLID sql.NullInt64
	var line_id int64

	_, err := db.ExecContext(ctx, queryCreateLine,
		sql.Named("return_value", sql.Out{Dest: &returnValue}),
		sql.Named("param_dept_id", dept_id),
		sql.Named("param_pl_desc", description),
		sql.Named("param_ext_link", externalLink),
		sql.Named("param_ext_info", extendedInfo),
		sql.Named("param_group_id", sg_id),
		sql.Named("param_user_id", userId),
		sql.Named("out_PL_Id", sql.Out{Dest: &outPLID}),
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to execute create logic: %w", err))
	}
	if returnValue.Int64 != 0 || outPLID.Int64 == 0 {
		return diag.FromErr(fmt.Errorf(
			"stored procedure returned failure status: return_value=%v, outPLID.Valid=%v, outPLID=%v",
			returnValue.Int64, outPLID.Valid, outPLID.Int64))
	}

	line_id = nullableInt64ToInt64(outPLID)

	d.Set("line_id", int(line_id))
	d.SetId(int64ToString(line_id))
	return resourceLineRead(ctx, d, m)
}

func resourceLineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := loadLinesCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}

	id := int64(d.Get("line_id").(int))
	line, ok := cachedLines[id]
	if !ok {
		d.SetId("")
		return nil
	}

	d.Set("id", id)
	d.Set("description", line.Description)
	d.Set("department", line.Department)
	d.Set("department_id", line.Dept_Id)
	d.Set("extended_info", line.ExtendedInfo)
	d.Set("external_link", line.ExternalLink)
	d.Set("security_group", line.SecurityGroup)
	d.Set("security_group_id", line.SecurityGroup_Id)
	return nil
}

func resourceLineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db := getDB(m)
	if err := loadLinesCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}
	description := d.Get("description").(string)
	dept_id := int64(d.Get("department_id").(int))
	extendedInfo := d.Get("extended_info").(string)
	sg_id := int64(d.Get("security_group_id").(int))
	externalLink := d.Get("external_link").(string)
	userId := 1

	var returnValue sql.NullInt64
	var outPLID sql.NullInt64

	_, err := db.ExecContext(ctx, queryUpdateLine,
		sql.Named("return_value", sql.Out{Dest: &returnValue}),
		sql.Named("param_dept_id", dept_id),
		sql.Named("param_pl_desc", description),
		sql.Named("param_ext_link", externalLink),
		sql.Named("param_ext_info", extendedInfo),
		sql.Named("param_group_id", sg_id),
		sql.Named("param_user_id", userId),
		sql.Named("out_PL_Id", sql.Out{Dest: &outPLID}),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceLineRead(ctx, d, m)
}

func resourceLineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	fmt.Println("Start of line delete")
	if err := loadLinesCache(ctx, m); err != nil {
		return diag.FromErr(err)
	}
	db := getDB(m)
	id := int64(d.Get("line_id").(int))
	userId := 1

	var returnValue int	
	_, err := db.ExecContext(ctx, queryDeleteLine,
		sql.Named("return_value", sql.Out{Dest: &returnValue}),
		sql.Named("param_pl_id", id),
		sql.Named("param_user_id", userId),
	)
	if err != nil {
		return diag.FromErr(err)
	}
	if returnValue != 0 {
		return diag.FromErr(fmt.Errorf("stored procedure returned failure status: %d", returnValue))
	}

	delete(cachedLines, id)
	d.SetId("")
	fmt.Println("End of line delete. pl_id: %d", id)
	return nil
}