package helper

import "fmt"

func ValidateField(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if len(value) > 200 {
		return fmt.Errorf("%s must not exceed 200 characters", field)
	}
	return nil
}
