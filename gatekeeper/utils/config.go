package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	InvalidRawValueErr = errors.New("invalid raw value. Must be either a string or bool")
	InvalidTypeErr     = errors.New("invalid type error")
)

type RequiredError struct {
	Flag string
}

func (e RequiredError) Error() string {
	return fmt.Sprintf("Missing required flag: %s", e.Flag)
}

// Coerce error represents a coercion error involving a parameter.
func NewCoerceError(field string, typ string, val interface{}) CoerceError {
	return CoerceError{
		Field: field,
		Type:  typ,
		Val:   val,
	}
}

type CoerceError struct {
	Field, Type string
	Val         interface{}
}

func (c CoerceError) Error() string {
	return fmt.Sprintf("Unabled to parse %s. Unable to coerce %v to %s", c.Field, c.Val, c.Type)
}

func ParseConfig(opts map[string]interface{}, config interface{}) error {
	val := reflect.ValueOf(config).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		flagName := field.Tag.Get("flag")
		if flagName == "" {
			continue
		}

		// check to see if the flag is required or not
		required := field.Tag.Get("required") != ""

		// find the value that _should_ be coerced. If the value is in
		// the options then that will be coerced, otherwise check if
		// the default tag is defined. If neither an opt or a default
		// is defined, then continue on
		rawValue, ok := opts[flagName]
		defaultValue := field.Tag.Get("default")

		// if the value was required and not passed in, emit an error
		if !ok && required {
			return RequiredError{flagName}
		}

		// if the flag is not required and no default is set, simply
		// continue with a no-op
		if !ok && defaultValue == "" {
			continue
		}

		// if the value wasn't found, use the default value
		if !ok {
			rawValue = defaultValue
		}

		// coerce the raw value into the config instance field; erring
		// out if unable to coerce.
		fieldVal := val.FieldByName(field.Name)
		coerced, err := coerce(field.Name, rawValue, fieldVal.Interface())

		// if an error occurs in the coerce phase, return from the function early bubbling the error up
		if err != nil {
			return err
		}
		fieldVal.Set(reflect.ValueOf(coerced))
	}

	return nil
}

func coerce(field string, rawVal interface{}, dest interface{}) (interface{}, error) {
	// the raw value can be either a string or a bool. If the value is a
	// bool then the interface destination must also be bool otherwise error out.
	if boolVal, ok := rawVal.(bool); ok {
		if _, ok := dest.(bool); !ok {
			return nil, NewCoerceError(field, "bool", rawVal)
		}
		return boolVal, nil
	}

	strVal, ok := rawVal.(string)
	if !ok {
		return nil, InvalidRawValueErr
	}

	// do a type switch against the destination to decipher what to coerce this string as
	switch dest.(type) {
	case string:
		return strVal, nil
	case time.Duration:
		val, err := time.ParseDuration(strVal)
		if err != nil {
			return nil, NewCoerceError(field, "time.Duration", strVal)
		}
		return val, nil
	case bool:
		val, err := strconv.ParseBool(strVal)
		if err != nil {
			return nil, NewCoerceError(field, "bool", strVal)
		}
		return val, nil
	case uint:
		val, err := strconv.ParseUint(strVal, 10, 64)
		if err != nil {
			return nil, NewCoerceError(field, "uint", strVal)
		}
		return uint(val), nil
	case int:
		val, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return nil, NewCoerceError(field, "int", strVal)
		}
		return int(val), nil
	case float64:
		val, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return nil, NewCoerceError(field, "float64", strVal)
		}
		return val, nil
	case []string:
		return strings.Split(strVal, ","), nil
	default:
		return nil, InvalidTypeErr
	}

	return nil, InvalidTypeErr
}
