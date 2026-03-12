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

package main

import (
	"context"

	"github.com/harness/liquibase-drone-plugin/internal/logger"
	"github.com/harness/liquibase-drone-plugin/plugin"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(logger.GetDefaultLoggerFormatterWithoutTimestamp())

	var args plugin.Args
	if err := envconfig.Process("", &args); err != nil {
		logrus.Fatalln(err)
	}

	level, err := logrus.ParseLevel(args.LogLevel)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Warnf("Invalid log level '%s', defaulting to info", args.LogLevel)
	} else {
		logrus.SetLevel(level)
	}

	if err := plugin.Exec(context.Background(), args); err != nil {
		logrus.Fatalln(err)
	}
}
