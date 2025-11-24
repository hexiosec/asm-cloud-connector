package util

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

// StringToTimeHookFunc returns a DecodeHookFunc that converts
// strings to time.Time. Support multiple time formats.
func StringToTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		// Convert it by parsing
		if parsedRFC, err := time.Parse(time.RFC3339, data.(string)); err == nil {
			return parsedRFC, nil
		} else if parsedDate, err := time.Parse(time.DateOnly, data.(string)); err == nil {
			return parsedDate, nil
		} else {
			return nil, fmt.Errorf("mapstructure: failed parsing time %v", data)
		}
	}
}

func MapStructDecode(data interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			DecodeHook: StringToTimeHookFunc(),
			Result:     result,
		})
	if err != nil {
		panic(err)
	}

	if err := decoder.Decode(data); err != nil {
		return err
	}

	return nil
}

func Validate(data interface{}) error {
	v := validator.New()

	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return fmt.Errorf("mapstructure: data must be a pointer")
	}

	var errs error = nil
	if reflect.ValueOf(data).Elem().Kind() == reflect.Slice {
		slice := reflect.ValueOf(data).Elem()
		for i := 0; i < slice.Len(); i++ {
			err := v.Struct(slice.Index(i).Interface())
			errs = errors.Join(errs, err)
		}
		return errs
	} else {
		return v.Struct(data)
	}
}

func MapStructDecodeAndValidate(data interface{}, result interface{}) error {
	if err := MapStructDecode(data, result); err != nil {
		return err
	}
	return Validate(result)
}
