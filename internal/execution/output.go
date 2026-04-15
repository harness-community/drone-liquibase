// Copyright 2025 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ExecutionStatus represents the status of plugin execution.
type ExecutionStatus string

const (
	ExecutionStatusSuccess ExecutionStatus = "SUCCESS"
	ExecutionStatusFailure ExecutionStatus = "FAILURE"
)

// OutputPropertyType represents the type of output property.
type OutputPropertyType string

const (
	OutputPropertyTypeSimple  OutputPropertyType = "simple"
	OutputPropertyTypeComplex OutputPropertyType = "complex"
	OutputPropertyTypeArray   OutputPropertyType = "array"
)

// Output represents the plugin execution output.
type Output struct {
	properties      map[string]interface{}
	propertyOrder   []string // Maintains insertion order
	executionStatus ExecutionStatus
	response        *Response
}

// Response represents an execution response.
type Response struct {
	FailureType string
	Message     string
}

// NewOutput creates a new Output instance.
func NewOutput() *Output {
	return &Output{
		properties: make(map[string]interface{}),
	}
}

// AddProperty adds a property to the output.
func (o *Output) AddProperty(name string, propType OutputPropertyType, value interface{}) {
	if _, exists := o.properties[name]; !exists {
		o.propertyOrder = append(o.propertyOrder, name)
	}
	o.properties[name] = value
}

// GetProperty returns a property value by name, or nil if not set.
func (o *Output) GetProperty(name string) interface{} {
	return o.properties[name]
}

// SetExecutionStatus sets the execution status.
func (o *Output) SetExecutionStatus(status ExecutionStatus) {
	o.executionStatus = status
}

// SetExecutionResponse sets the execution response.
func (o *Output) SetExecutionResponse(resp Response) {
	o.response = &resp
}

// SetExecutionFailureType sets the failure type.
func (o *Output) SetExecutionFailureType(failureType string) {
	if o.response == nil {
		o.response = &Response{}
	}
	o.response.FailureType = failureType
}

// CreateOutputFile writes the output to the specified file in DRONE_OUTPUT format.
func (o *Output) CreateOutputFile(filePath string) (string, error) {
	var lines []string

	// Add all properties in insertion order
	for _, name := range o.propertyOrder {
		value := o.properties[name]
		switch v := value.(type) {
		case string:
			lines = append(lines, fmt.Sprintf("%s=%s", name, v))
		default:
			// For complex types, JSON encode
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal property %s: %w", name, err)
			}
			lines = append(lines, fmt.Sprintf("%s=%s", name, string(jsonBytes)))
		}
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write output file: %w", err)
	}

	return filePath, nil
}

// HandleError processes an error and populates the response.
func HandleError(err error, resp *Response) error {
	if err == nil {
		return nil
	}
	resp.Message = err.Error()
	resp.FailureType = "UNKNOWN_ERROR"
	return err
}

// ErrorToString converts an error to string representation.
func ErrorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
