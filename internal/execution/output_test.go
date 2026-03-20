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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewOutput(t *testing.T) {
	output := NewOutput()
	if output == nil {
		t.Fatal("NewOutput() returned nil")
	}
	if output.properties == nil {
		t.Error("properties map should be initialized")
	}
}

func TestAddProperty(t *testing.T) {
	output := NewOutput()
	output.AddProperty("test_key", OutputPropertyTypeSimple, "test_value")

	if output.properties["test_key"] != "test_value" {
		t.Errorf("AddProperty() property value = %v, want %v", output.properties["test_key"], "test_value")
	}
}

func TestSetExecutionStatus(t *testing.T) {
	output := NewOutput()
	output.SetExecutionStatus(ExecutionStatusSuccess)

	if output.executionStatus != ExecutionStatusSuccess {
		t.Errorf("SetExecutionStatus() = %v, want %v", output.executionStatus, ExecutionStatusSuccess)
	}
}

func TestSetExecutionResponse(t *testing.T) {
	output := NewOutput()
	resp := Response{
		FailureType: "TEST_ERROR",
		Message:     "test message",
	}
	output.SetExecutionResponse(resp)

	if output.response == nil {
		t.Fatal("response should not be nil")
	}
	if output.response.FailureType != "TEST_ERROR" {
		t.Errorf("FailureType = %v, want %v", output.response.FailureType, "TEST_ERROR")
	}
}

func TestCreateOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.out")

	output := NewOutput()
	output.AddProperty("exit_code", OutputPropertyTypeSimple, "0")
	output.AddProperty("status", OutputPropertyTypeSimple, "success")

	path, err := output.CreateOutputFile(outputFile)
	if err != nil {
		t.Fatalf("CreateOutputFile() error = %v", err)
	}

	if path != outputFile {
		t.Errorf("CreateOutputFile() path = %v, want %v", path, outputFile)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check that both properties are in the file
	if !strings.Contains(string(content), "exit_code=0") {
		t.Errorf("Output file missing exit_code=0")
	}
	if !strings.Contains(string(content), "status=success") {
		t.Errorf("Output file missing status=success")
	}
}

func TestCreateOutputFilePreservesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.out")

	output := NewOutput()
	output.AddProperty("exit_code", OutputPropertyTypeSimple, "0")
	output.AddProperty("step_output", OutputPropertyTypeComplex, map[string]string{"key": "value"})

	_, err := output.CreateOutputFile(outputFile)
	if err != nil {
		t.Fatalf("CreateOutputFile() error = %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// exit_code should be first (insertion order)
	if !strings.HasPrefix(lines[0], "exit_code=") {
		t.Errorf("First line should be exit_code, got: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "step_output=") {
		t.Errorf("Second line should be step_output, got: %s", lines[1])
	}
}

func TestHandleError(t *testing.T) {
	resp := &Response{}
	err := HandleError(nil, resp)
	if err != nil {
		t.Errorf("HandleError(nil) should return nil")
	}

	testErr := &testError{msg: "test error"}
	err = HandleError(testErr, resp)
	if err == nil {
		t.Error("HandleError() should return error")
	}
	if resp.Message != "test error" {
		t.Errorf("resp.Message = %v, want %v", resp.Message, "test error")
	}
	if resp.FailureType != "UNKNOWN_ERROR" {
		t.Errorf("resp.FailureType = %v, want %v", resp.FailureType, "UNKNOWN_ERROR")
	}
}

func TestErrorToString(t *testing.T) {
	if ErrorToString(nil) != "" {
		t.Error("ErrorToString(nil) should return empty string")
	}

	err := &testError{msg: "test"}
	if ErrorToString(err) != "test" {
		t.Errorf("ErrorToString() = %v, want %v", ErrorToString(err), "test")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
