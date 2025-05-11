package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	_ "github.com/microsoft/go-mssqldb"
)

func resourceUnit() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUnitCreate,
		ReadContext:   resourceUnitRead,
		UpdateContext: resourceUnitUpdate,
		DeleteContext: resourceUnitDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"pu_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func getConnection(d *schema.ResourceData, m interface{}) (*sql.DB, error) {
	db, ok := m.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("failed to get database connection from provider metadata")
	}
	return db, nil
}

func resourceUnitCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db, err := getConnection(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer db.Close()

	puID := d.Get("pu_id").(int)
	description := d.Get("description").(string)

	_, err = db.ExecContext(ctx,
		"INSERT INTO SOADB.dbo.Local_Units (PU_Id, Description) VALUES (@p1, @p2)",
		puID, description)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(puID))
	return resourceUnitRead(ctx, d, m)
}

func resourceUnitRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db, err := getConnection(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer db.Close()

	var diags diag.Diagnostics
	puID, _ := strconv.Atoi(d.Id())

	row := db.QueryRowContext(ctx,
		"SELECT PU_Id, Description FROM SOADB.dbo.Local_Units WHERE PU_Id = @p1",
		puID)

	var description string
	err = row.Scan(&puID, &description)
	if err == sql.ErrNoRows {
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("pu_id", puID)
	d.Set("description", description)

	return diags
}

func resourceUnitUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db, err := getConnection(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer db.Close()

	puID, _ := strconv.Atoi(d.Id())
	description := d.Get("description").(string)

	_, err = db.ExecContext(ctx,
		"UPDATE SOADB.dbo.Local_Units SET Description = @p1 WHERE PU_Id = @p2",
		description, puID)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceUnitRead(ctx, d, m)
}

func resourceUnitDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	db, err := getConnection(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer db.Close()

	puID, _ := strconv.Atoi(d.Id())

	_, err = db.ExecContext(ctx,
		"DELETE FROM SOADB.dbo.Local_Units WHERE PU_Id = @p1",
		puID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
