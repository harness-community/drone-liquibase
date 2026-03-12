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

package logger

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// SimpleFormatter is a basic formatter without timestamp.
type SimpleFormatter struct{}

// Format formats the log entry.
func (f *SimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())
	msg := fmt.Sprintf("[%s] %s\n", level, entry.Message)
	return []byte(msg), nil
}

// GetDefaultLoggerFormatterWithoutTimestamp returns a simple formatter.
func GetDefaultLoggerFormatterWithoutTimestamp() logrus.Formatter {
	return &SimpleFormatter{}
}
