package helper

import (
	"fmt"

	"github.com/aws/smithy-go"
)

type MockAPIError struct {
	Code string
}

func (e *MockAPIError) ErrorCode() string {
	return e.Code
}

func (e *MockAPIError) ErrorMessage() string {
	return ""
}

func (e *MockAPIError) ErrorFault() smithy.ErrorFault {
	return 0
}

func (e *MockAPIError) Error() string {
	return fmt.Sprintf("mock api error %s", e.Code)
}

func NewMockAPIError(code string) *MockAPIError {
	return &MockAPIError{
		Code: code,
	}
}
