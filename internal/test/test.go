// Copyright 2021 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/treydock/subid-ldap/internal/config"
)

func TestConfig() config.Config {
	return config.Config{
		SubIDStart: 65537,
		SubIDRange: 65536,
	}
}

func GetFixture(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixturesDir := filepath.Join(dir, "fixtures")
	return filepath.Join(fixturesDir, name)
}

func CreateTmpFile(name string, logger *slog.Logger) (string, error) {
	file, err := os.CreateTemp("", name)
	if err != nil {
		logger.Error("Error creating temp file", "err", err)
		return "", err
	}
	defer file.Close()
	return file.Name(), nil
}

func CreateSubUIDFixture(name string) (string, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fixture := GetFixture(name)
	tmpFile, err := CreateTmpFile(name, logger)
	if err != nil {
		return "", err
	}
	logger.Debug("Loading fixture", "fixture", fixture)
	content, err := os.ReadFile(fixture)
	if err != nil {
		logger.Error("Error reading fixture file", "err", err)
		return "", err
	}
	logger.Debug("Write fixture to tmp", "fixture", fixture, "dest", tmpFile, "content", string(content))
	err = os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		logger.Error("Error writing file", "err", err)
		return "", err
	}
	return tmpFile, nil
}
