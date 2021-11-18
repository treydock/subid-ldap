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
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/treydock/subid-ldap/internal/config"
	"github.com/treydock/subid-ldap/internal/metrics"
	"github.com/treydock/subid-ldap/internal/utils"
)

const (
	subidMode = 0644
)

var (
	SubUIDPath = "/etc/subuid"
	SubGIDPath = "/etc/subgid"
	maxID      = math.Pow(2, 32) - 1
)

type SubIDEntry struct {
	UID   string
	ID    int
	Count int
}

type SubID *map[int]SubIDEntry

func SubIDHeader(c *config.Config) string {
	return fmt.Sprintf("# Managed by %s: start=%d range=%d", config.AppName, c.SubIDStart, c.SubIDRange)
}

func SubIDKeys(input SubID) []int {
	keys := make([]int, 0, len(*input))
	for key := range *input {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func SubIDGenerate(config *config.Config, logger log.Logger) SubID {
	length := ((maxID - float64(config.SubIDStart)) / float64(config.SubIDRange))
	level.Debug(logger).Log("msg", "Generate entries",
		"length", int64(math.Floor(length)), "max", maxID, "start", config.SubIDStart, "range", config.SubIDRange)
	entries := make(map[int]SubIDEntry, int(math.Floor(length)))
	for i := config.SubIDStart; i < int(maxID); i = i + config.SubIDRange + 1 {
		entry := SubIDEntry{
			ID:    i,
			Count: config.SubIDRange,
		}
		entries[i] = entry
	}
	return &entries
}

func SubIDManaged(path string, c *config.Config, logger log.Logger) (bool, error) {
	if exists, err := utils.Exists(path); err != nil {
		level.Error(logger).Log("msg", "Unable to check if subid exists", "err", err)
		return false, err
	} else if !exists {
		return true, nil
	}
	level.Debug(logger).Log("msg", "Read subid file", "path", path)
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return false, nil
	}
	header := SubIDHeader(c)
	level.Debug(logger).Log("msg", "Check if line is managed", "line", lines[0], "header", header)
	if strings.HasPrefix(lines[0], "#") && strings.Contains(lines[0], header) {
		return true, nil
	}
	return false, nil
}

func SubIDLoad(path string, logger log.Logger) (SubID, error) {
	entries := make(map[int]SubIDEntry)
	content, err := os.ReadFile(path)
	if err != nil {
		return &entries, err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		items := strings.Split(line, ":")
		if len(items) != 3 {
			level.Debug(logger).Log("msg", "Skipping line that does not contain 3 items", "line", line)
			continue
		}
		id, err := strconv.Atoi(items[1])
		if err != nil {
			level.Error(logger).Log("msg", "Unable to parse ID integer", "line", line, "err", err)
			continue
		}
		count, err := strconv.Atoi(items[2])
		if err != nil {
			level.Error(logger).Log("msg", "Unable to parse count integer", "line", line, "err", err)
			continue
		}
		entry := SubIDEntry{
			UID:   items[0],
			ID:    id,
			Count: count,
		}
		entries[id] = entry
	}
	return &entries, nil
}

func SubIDSaveNew(users []string, path string, c *config.Config) error {
	metrics.MetricSubIDTotal.Set(float64(len(users)))
	metrics.MetricSubIDAdded.Set(float64(len(users)))
	lines := []string{SubIDHeader(c)}
	id := c.SubIDStart
	for _, user := range users {
		line := fmt.Sprintf("%s:%d:%d", user, id, c.SubIDRange)
		lines = append(lines, line)
		id = id + c.SubIDRange + 1
	}
	content := []byte(strings.Join(lines, "\n"))
	err := os.WriteFile(path, content, 0644)
	if err != nil {
		return err
	}
	return nil
}

func SubIDUpdate(users []string, existing SubID, subids SubID, path string, c *config.Config, logger log.Logger) error {
	metrics.MetricSubIDTotal.Set(float64(len(users)))
	var added, removed float64
	// Remove users from existing if no longer valid
	for id, e := range *existing {
		if !utils.SliceContains(users, e.UID) {
			level.Debug(logger).Log("msg", "Remove UID from subids", "uid", e.UID)
			e.UID = ""
			(*existing)[id] = e
			removed++
		}
	}

	// Add existing
	existingUsers := []string{}
	for id, e := range *existing {
		level.Debug(logger).Log("msg", "Adding existing subid", "uid", e.UID, "id", id)
		existingUsers = append(existingUsers, e.UID)
		(*subids)[id] = e
	}

	// Get UIDs to add
	newUIDs := []string{}
	for _, user := range users {
		if utils.SliceContains(existingUsers, user) {
			continue
		}
		newUIDs = append(newUIDs, user)
	}

	// Get unassigned IDs
	unassignedIDs := []int{}
	for _, id := range SubIDKeys(subids) {
		s := (*subids)[id]
		if s.UID == "" {
			unassignedIDs = append(unassignedIDs, id)
		}
	}
	level.Debug(logger).Log("unassignedIDs", len(unassignedIDs))

	//Add users
	for i, uid := range newUIDs {
		level.Debug(logger).Log("msg", "Add users", "i", i, "unassignedIDs", len(unassignedIDs), "uid", uid)
		if i >= len(unassignedIDs) {
			level.Error(logger).Log("msg", "Insufficient subids available", "uid", uid)
			metrics.MetricError.Set(1)
			break
		}
		id := unassignedIDs[i]
		s := (*subids)[id]
		level.Debug(logger).Log("msg", "Adding user subid", "uid", uid, "id", id)
		s.UID = uid
		(*subids)[id] = s
		added++
	}

	level.Debug(logger).Log("msg", "Subids processed", "added", added, "removed", removed)
	metrics.MetricSubIDAdded.Set(added)
	metrics.MetricSubIDRemoved.Set(removed)

	lines := []string{SubIDHeader(c)}
	for _, id := range SubIDKeys(subids) {
		if (*subids)[id].UID == "" {
			continue
		}
		line := fmt.Sprintf("%s:%d:%d", (*subids)[id].UID, (*subids)[id].ID, (*subids)[id].Count)
		lines = append(lines, line)
	}
	content := []byte(strings.Join(lines, "\n"))
	level.Debug(logger).Log("msg", "Update subid file", "path", path)
	err := os.WriteFile(path, content, subidMode)
	if err != nil {
		return err
	}
	return nil
}

func SubGIDSave(subuid string, subgid string) error {
	content, err := os.ReadFile(subuid)
	if err != nil {
		return err
	}
	err = os.WriteFile(subgid, content, subidMode)
	if err != nil {
		return err
	}
	return nil
}
