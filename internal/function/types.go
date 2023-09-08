package function

import (
	"fmt"
	"reflect"
	"strconv"
)

type DataTypeEnum interface {
	// VariantName() string
	TypeCheck(val interface{}) error
}

// we do not want a generic Object type: it could lead to bugs and makes harder the conversion

// None represent a nil value
// type None struct{}

// Text represent a string value
type Text struct{}

// Int represent an int value
type Int struct{}

// Float represent a float64 or float32 value
type Float struct{}

// Bool represent a boolean value
type Bool struct{}

// Array represents an array of one of the dataTypes
type Array[D DataTypeEnum] struct {
	DataType D
}

// Option represent a value that can either be a DataType or be nil
// type Option[D DataTypeEnum, N None] struct{}

// TODO: be sure that the input it is not always string. If it is always string, use string instead of interface{}
func (t Text) TypeCheck(val interface{}) error {
	switch val.(type) {
	case string:
		return nil
	default:
		return fmt.Errorf("val should be Text, but is %v", val)
	}
}
func (i Int) TypeCheck(val interface{}) error {
	switch val.(type) {
	case int:
		return nil
	case string:
		_, err := strconv.Atoi(val.(string))
		if err == nil {
			return nil
		}
		return fmt.Errorf("val is a string '%s', but cannot be cast to an Int", val.(string))
	default:
		return fmt.Errorf("val should be Int, but is %v", val)
	}
}

func (b Bool) TypeCheck(val interface{}) error {
	switch val.(type) {
	case bool:
		return nil
	case int:
		if val.(int) == 1 || val.(int) == 0 {
			return nil
		}
		return fmt.Errorf("val is of type int, but cannot be converted to bool")
	case string:
		v := val.(string)
		if v == "false" || v == "False" || v == "true" || v == "True" || v == "1" || v == "0" {
			return nil
		}
		return fmt.Errorf("val is of type string, but cannot be converted to bool")
	default:
		return fmt.Errorf("val should be Bool, but is %v", val)
	}
}

func (f Float) TypeCheck(val interface{}) error {
	switch val.(type) {
	case float64:
		return nil
	case float32:
		return nil
	case string:
		_, err := strconv.ParseFloat(val.(string), 32)
		_, err2 := strconv.ParseFloat(val.(string), 64)
		if err == nil || err2 == nil {
			return nil
		}
		return fmt.Errorf("val is a string '%s', but cannot be cast to a Float", val.(string))
	default:
		return fmt.Errorf("val should be Float but is %v", val)
	}
}

// TypeCheck represents an array of one of the dataTypes
func (a Array[D]) TypeCheck(val interface{}) error {
	switch reflect.TypeOf(val).Kind() {
	case reflect.Slice:
		// convert interface{} to []interface{}
		var genericSlice []interface{}
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Slice {
			for i := 0; i < rv.Len(); i++ {
				genericSlice = append(genericSlice, rv.Index(i).Interface())
			}
		}

		typeError := ""
		for i, t := range genericSlice {
			err := a.DataType.TypeCheck(t)
			if err != nil {
				typeError += fmt.Sprintf("\ntype-error: element %d of slice has wrong type", i)
				break
			}
		}
		if typeError != "" {
			return fmt.Errorf("%s", typeError)
		}
		return nil
	default:
		fmt.Printf("name of type: %s\n", reflect.TypeOf(val).Name())
		typeElem := reflect.TypeOf(val).Elem()
		err := fmt.Errorf("val should be a slice, but is %v of type %s", val, typeElem.Name())
		return err
	}
}
