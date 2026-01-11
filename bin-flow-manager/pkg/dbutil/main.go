package dbutil

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
)

// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string {
	return getDBFieldsRecursive(reflect.ValueOf(model))
}

// getDBFieldsRecursive is the internal recursive function that works with reflect.Value
func getDBFieldsRecursive(val reflect.Value) []string {
	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Must be a struct at this point
	if val.Kind() != reflect.Struct {
		return []string{}
	}

	typ := val.Type()
	fields := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedFields := getDBFieldsRecursive(embeddedVal)
			fields = append(fields, embeddedFields...)
			continue
		}

		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		columnName := parts[0]

		fields = append(fields, columnName)
	}

	return fields
}

// PrepareValues converts struct fields to database values for INSERT/UPDATE
func PrepareValues(model interface{}) ([]interface{}, error) {
	return prepareValuesRecursive(reflect.ValueOf(model))
}

// prepareValuesRecursive is the internal recursive implementation
func prepareValuesRecursive(val reflect.Value) ([]interface{}, error) {
	// Handle pointer dereferencing
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Return empty slice for non-struct types
	if val.Kind() != reflect.Struct {
		return []interface{}{}, nil
	}

	typ := val.Type()
	values := []interface{}{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedValues, err := prepareValuesRecursive(embeddedVal)
			if err != nil {
				return nil, err
			}
			values = append(values, embeddedValues...)
			continue
		}

		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		// Parse tag: "column_name" or "column_name,conversion_type"
		parts := strings.Split(tag, ",")
		conversionType := ""
		if len(parts) > 1 {
			conversionType = parts[1]
		}

		fieldVal := val.Field(i)

		// Apply conversions based on type
		switch conversionType {
		case "uuid":
			// Convert uuid.UUID to []byte
			if fieldVal.Type() == reflect.TypeOf(uuid.UUID{}) {
				uuidVal := fieldVal.Interface().(uuid.UUID)
				values = append(values, uuidVal.Bytes())
			} else {
				return nil, fmt.Errorf("field %s: expected uuid.UUID type for uuid conversion", field.Name)
			}
		case "json":
			// Convert to JSON string
			jsonBytes, err := json.Marshal(fieldVal.Interface())
			if err != nil {
				return nil, fmt.Errorf("field %s: cannot marshal to JSON: %w", field.Name, err)
			}
			values = append(values, string(jsonBytes))
		default:
			values = append(values, fieldVal.Interface())
		}
	}

	return values, nil
}

// fieldScanTarget represents a field's scan metadata
type fieldScanTarget struct {
	fieldVal   *reflect.Value
	scanTarget interface{}
}

// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
func ScanRow(row *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	destVal = destVal.Elem()
	if destVal.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	// Build list of scan targets with NULL handling
	scanTargets := buildScanTargetsRecursive(destVal)

	// Extract just the scan interfaces for row.Scan
	scanInterfaces := make([]interface{}, len(scanTargets))
	for i, target := range scanTargets {
		scanInterfaces[i] = target.scanTarget
	}

	// Scan the row
	if err := row.Scan(scanInterfaces...); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Copy values from sql.Null* types to actual fields
	for _, target := range scanTargets {
		if err := copyFromNullType(target.scanTarget, target.fieldVal); err != nil {
			return fmt.Errorf("copy from null type failed: %w", err)
		}
	}

	return nil
}

// buildScanTargetsRecursive recursively builds scan targets for embedded structs
func buildScanTargetsRecursive(val reflect.Value) []fieldScanTarget {
	// Handle pointer dereferencing
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return []fieldScanTarget{}
	}

	typ := val.Type()
	scanTargets := []fieldScanTarget{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Handle embedded structs recursively
		if field.Anonymous {
			embeddedVal := val.Field(i)
			embeddedTargets := buildScanTargetsRecursive(embeddedVal)
			scanTargets = append(scanTargets, embeddedTargets...)
			continue
		}

		tag := field.Tag.Get("db")

		// Skip fields without db tag or with "-"
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := val.Field(i)

		// Create appropriate sql.Null* type based on field type
		scanTarget := createNullScanTarget(fieldVal)

		// Store both the field reference and scan target
		fieldValCopy := fieldVal
		scanTargets = append(scanTargets, fieldScanTarget{
			fieldVal:   &fieldValCopy,
			scanTarget: scanTarget,
		})
	}

	return scanTargets
}

// createNullScanTarget creates appropriate sql.Null* type for a field
func createNullScanTarget(fieldVal reflect.Value) interface{} {
	switch fieldVal.Kind() {
	case reflect.String:
		return new(sql.NullString)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(sql.NullInt64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(sql.NullInt64)
	case reflect.Float32, reflect.Float64:
		return new(sql.NullFloat64)
	case reflect.Bool:
		return new(sql.NullBool)
	default:
		// For complex types (UUID, JSON), scan directly
		return fieldVal.Addr().Interface()
	}
}

// copyFromNullType copies value from sql.Null* type to field if valid
func copyFromNullType(scanTarget interface{}, fieldVal *reflect.Value) error {
	switch v := scanTarget.(type) {
	case *sql.NullString:
		if v.Valid {
			fieldVal.SetString(v.String)
		}
		// If NULL, leave field as zero value (empty string)
	case *sql.NullInt64:
		if v.Valid {
			// Convert to appropriate int type
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldVal.SetInt(v.Int64)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldVal.SetUint(uint64(v.Int64))
			}
		}
		// If NULL, leave field as zero value (0)
	case *sql.NullFloat64:
		if v.Valid {
			fieldVal.SetFloat(v.Float64)
		}
		// If NULL, leave field as zero value (0.0)
	case *sql.NullBool:
		if v.Valid {
			fieldVal.SetBool(v.Bool)
		}
		// If NULL, leave field as zero value (false)
	default:
		// For complex types, the value was scanned directly - no copy needed
	}
	return nil
}
