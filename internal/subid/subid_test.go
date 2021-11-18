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

package subid

import (
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/treydock/subid-ldap/internal/test"
)

func TestSubIDGenerate(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	c := test.TestConfig()
	subids := SubIDGenerate(&c, logger)
	if len(*subids) != 65534 {
		t.Errorf("Unexpected length for generated subid entries, got: %d", len(*subids))
	}
	if _, ok := (*subids)[65537]; !ok {
		t.Errorf("Expected starting subid to be 65537")
	}
}

func TestSubIDGenerateCustomStartAndRange(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	c := test.TestConfig()
	c.SubIDStart = 1000000
	c.SubIDRange = 100000
	subids := SubIDGenerate(&c, logger)
	if len(*subids) != 42940 {
		t.Errorf("Unexpected length for generated subid entries, got: %d", len(*subids))
	}
	if _, ok := (*subids)[1000000]; !ok {
		t.Errorf("Expected starting subid to be 1000000")
	}
}

func TestSubIDManaged(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	fixture, err := test.CreateSubUIDFixture("subuid1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer os.Remove(fixture)
	c := test.TestConfig()
	managed, err := SubIDManaged(fixture, &c, logger)
	if err != nil {
		t.Errorf("Unexpected error %s", err)
		return
	}
	if !managed {
		t.Errorf("File should be managed, not unmanaged")
	}
}

func TestSubIDUnManaged(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	fixture, err := test.CreateSubUIDFixture("subuid1-unmanaged")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer os.Remove(fixture)
	c := test.TestConfig()
	managed, err := SubIDManaged(fixture, &c, logger)
	if err != nil {
		t.Errorf("Unexpected error %s", err)
		return
	}
	if managed {
		t.Errorf("File should be unmanaged, not managed")
	}
	tmp, err := test.CreateTmpFile("subuid", logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer os.Remove(tmp)
	managed, err = SubIDManaged(tmp, &c, logger)
	if err != nil {
		t.Errorf("Unexpected error %s", err)
		return
	}
	if managed {
		t.Errorf("File should be unmanaged, not managed")
	}
}

func TestSubIDManagedErrors(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	c := test.TestConfig()
	managed, err := SubIDManaged("/dne", &c, logger)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
	if !managed {
		t.Errorf("File should be managed, not unmanaged")
	}
}

func TestSubIDLoad(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	fixture, err := test.CreateSubUIDFixture("subuid1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer os.Remove(fixture)
	subids, err := SubIDLoad(fixture, logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(*subids) != 3 {
		t.Errorf("Unexpected number of subids returned, got %d", len(*subids))
		return
	}
	if val := (*subids)[65537].UID; val != "1000" {
		t.Errorf("Unexpected value for UID, got: %s", val)
	}
}

func TestSubIDLoadErrors(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	_, err := SubIDLoad("/dne", logger)
	if err == nil {
		t.Fatal("Expected an error")
	}
	fixture, err := test.CreateSubUIDFixture("subuid-bad")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer os.Remove(fixture)
	subids, err := SubIDLoad(fixture, logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(*subids) != 1 {
		t.Errorf("Unexpected number of subids returned, got %d", len(*subids))
		return
	}
	if val := (*subids)[196611].UID; val != "1003" {
		t.Errorf("Unexpected value for UID, got: %s", val)
	}
}

func TestSubIDSaveNew(t *testing.T) {
	tmp, err := os.CreateTemp("", "subuid")
	if err != nil {
		t.Errorf("Error creating temp file: %s", err)
		return
	}
	defer os.Remove(tmp.Name())
	users := []string{"1000", "1002", "1003"}
	c := test.TestConfig()
	err = SubIDSaveNew(users, tmp.Name(), &c)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	subids, err := SubIDLoad(tmp.Name(), logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(*subids) != 3 {
		t.Errorf("Unexpected subid count, got %d", len(*subids))
	}
	if val := (*subids)[65537].UID; val != "1000" {
		t.Errorf("Unexpected value for UID, got %s", val)
	}
}

func TestSubIDSaveNewErrors(t *testing.T) {
	users := []string{"1000", "1002", "1003"}
	c := test.TestConfig()
	err := SubIDSaveNew(users, "/dne/test", &c)
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestSubIDUpdate(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	tmp, err := test.CreateTmpFile("subuid", logger)
	if err != nil {
		t.Errorf("Error creating temp file: %s", err)
		return
	}
	defer os.Remove(tmp)
	users := []string{"1000", "1001", "1002"}
	fixture, err := test.CreateSubUIDFixture("subuid1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	existing, err := SubIDLoad(fixture, logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	c := test.TestConfig()
	subids := SubIDGenerate(&c, logger)
	err = SubIDUpdate(users, existing, subids, tmp, &c, logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	savedVal, err := SubIDLoad(tmp, logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(*savedVal) != 3 {
		t.Errorf("Unexpected subid count, got %d\n%+v", len(*savedVal), *savedVal)
	}
	if val := (*savedVal)[65537].UID; val != "1000" {
		t.Errorf("Unexpected value for UID, got %s\n%+v", val, *savedVal)
	}
	if val := (*savedVal)[131074].UID; val != "1001" {
		t.Errorf("Unexpected value for UID, got %s", val)
	}
	if val := (*savedVal)[196611].UID; val != "1002" {
		t.Errorf("Unexpected value for UID, got %s\n%+v", val, *savedVal)
	}
}

func TestSubIDUpdateError(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	c := test.TestConfig()
	subids := SubIDGenerate(&c, logger)
	users := []string{"1000", "1001", "1002"}
	existing, _ := SubIDLoad("/dne/test", logger)
	err := SubIDUpdate(users, existing, subids, "/dne/test", &c, logger)
	if err == nil {
		t.Errorf("Expected an error")
	}
	oldMaxID := maxID
	maxID = float64(c.SubIDStart * 2)
	subids = SubIDGenerate(&c, logger)
	maxID = oldMaxID
	tmp, err := test.CreateTmpFile("subuid", logger)
	if err != nil {
		t.Errorf("Error creating temp file: %s", err)
		return
	}
	defer os.Remove(tmp)
	existing, _ = SubIDLoad(tmp, logger)
	err = SubIDUpdate(users, existing, subids, tmp, &c, logger)
	if err != nil {
		t.Errorf("Unexpected an error: %s", err)
	}
	if len(*subids) != 1 {
		t.Errorf("Unexpected number of subids, got %d", len(*subids))
	}
}

func TestSubGIDSaveError(t *testing.T) {
	err := SubGIDSave("/dne/subuid", "/dne/subgid")
	if err == nil {
		t.Errorf("Expected an error")
	}
	tmp, err := test.CreateTmpFile("subuid", log.NewNopLogger())
	if err != nil {
		t.Errorf("Error creating temp file: %s", err)
		return
	}
	defer os.Remove(tmp)
	err = SubGIDSave(tmp, "/dne/subgid")
	if err == nil {
		t.Errorf("Expected an error")
	}
}
