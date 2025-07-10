package main

import (
	"regexp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func validateTitle() schema.SchemaValidateFunc {
	pattern := regexp.MustCompile(`^[\w\-\(\)]+( [\w\-\(\)]+)*$`)

	return func(val interface{}, key string) (warns []string, errs []error) {
		v := val.(string)

		if len(v) > 50 {
			errs = append(errs, fmt.Errorf("%q must be 50 characters or fewer", key))
		}

		if !pattern.MatchString(v) {
			errs = append(errs, fmt.Errorf("%q can only contain alphanumeric characters, spaces, dashes (-), underscores (_), and parentheses (), and must not start or end with spaces", key))
		}

		return
	}
}


func validateVarchar255() schema.SchemaValidateFunc {
	return func(val interface{}, key string) (warns []string, errs []error) {
		v := val.(string)
		if len(v) > 255 {
			errs = append(errs, fmt.Errorf("%q must be less than or equal to 255 characters, or empty", key))
		}
		return
	}
}

func validateTimeZone() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"Eastern Standard Time", "Pacific Standard Time", "UTC",
	}, false)
}