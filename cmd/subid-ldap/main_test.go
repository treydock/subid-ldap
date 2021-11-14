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

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/treydock/subid-ldap/internal/metrics"
	"github.com/treydock/subid-ldap/internal/test"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	ldapserver = "127.0.0.1:10389"
)

var (
	baseArgs = []string{
		fmt.Sprintf("--ldap.url=ldap://%s", ldapserver),
		fmt.Sprintf("--ldap.user-base-dn=%s", test.UserBaseDN),
		fmt.Sprintf("--ldap.bind-dn=%s", test.BindDN),
		"--ldap.bind-password=password",
	}
)

func TestMain(m *testing.M) {
	if _, err := kingpin.CommandLine.Parse(baseArgs); err != nil {
		os.Exit(1)
	}

	server := test.LdapServer()
	go func() {
		err := server.ListenAndServe(ldapserver)
		if err != nil {
			os.Exit(1)
		}
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestRun(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	subuid, err := test.CreateTmpFile("subuid", logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	defer os.Remove(subuid)
	subgid, err := test.CreateTmpFile("subgid", logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	defer os.Remove(subgid)
	args := append([]string{
		fmt.Sprintf("--subid.subuid=%s", subuid),
		fmt.Sprintf("--subid.subgid=%s", subgid),
		fmt.Sprintf("--ldap.user-filter=%s", test.UserFilter),
	}, baseArgs...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	err = run(logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	subuidContent, err := os.ReadFile(subuid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subgidContent, err := os.ReadFile(subgid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedSubUID := `# File managed by subid-ldap
1000:65537:65536
1001:131074:65536
1002:196611:65536
1003:262148:65536`
	if string(subuidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subuidContent), expectedSubUID)
	}
	if string(subgidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subgidContent), expectedSubUID)
	}

	expected := `
	# HELP subid_ldap_error Indicates an error was encountered
	# TYPE subid_ldap_error gauge
	subid_ldap_error 0
	# HELP subid_ldap_subid_added Number of subid entries added
	# TYPE subid_ldap_subid_added gauge
	subid_ldap_subid_added 4
	# HELP subid_ldap_subid_removed Number of subid entries removed
	# TYPE subid_ldap_subid_removed gauge
	subid_ldap_subid_removed 0
	# HELP subid_ldap_subid_total Total number of subid entries
	# TYPE subid_ldap_subid_total gauge
	subid_ldap_subid_total 4
	`

	if err := testutil.GatherAndCompare(metrics.MetricGathers(), strings.NewReader(expected),
		"subid_ldap_error", "subid_ldap_subid_added", "subid_ldap_subid_removed", "subid_ldap_subid_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

	args = append([]string{
		fmt.Sprintf("--subid.subuid=%s", subuid),
		fmt.Sprintf("--subid.subgid=%s", subgid),
		fmt.Sprintf("--ldap.user-filter=%s", test.UserFilterStatus),
	}, baseArgs...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	err = run(logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subuidContent, err = os.ReadFile(subuid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subgidContent, err = os.ReadFile(subgid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedSubUID = `# File managed by subid-ldap
1000:65537:65536
1001:131074:65536
1002:196611:65536`
	if string(subuidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subuidContent), expectedSubUID)
	}
	if string(subgidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subgidContent), expectedSubUID)
	}
}

func TestRunExisting(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	subuid, err := test.CreateSubUIDFixture("subuid1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	defer os.Remove(subuid)
	subgid, err := test.CreateTmpFile("subgid", logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	defer os.Remove(subgid)
	args := append([]string{
		fmt.Sprintf("--subid.subuid=%s", subuid),
		fmt.Sprintf("--subid.subgid=%s", subgid),
		fmt.Sprintf("--ldap.user-filter=%s", test.UserFilter),
	}, baseArgs...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	err = run(logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	subuidContent, err := os.ReadFile(subuid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subgidContent, err := os.ReadFile(subgid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedSubUID := `# File managed by subid-ldap
1000:65537:65536
1001:131074:65536
1003:196611:65536
1002:262148:65536`
	if string(subuidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subuidContent), expectedSubUID)
	}
	if string(subgidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subgidContent), expectedSubUID)
	}

	expected := `
	# HELP subid_ldap_error Indicates an error was encountered
	# TYPE subid_ldap_error gauge
	subid_ldap_error 0
	# HELP subid_ldap_subid_added Number of subid entries added
	# TYPE subid_ldap_subid_added gauge
	subid_ldap_subid_added 1
	# HELP subid_ldap_subid_removed Number of subid entries removed
	# TYPE subid_ldap_subid_removed gauge
	subid_ldap_subid_removed 0
	# HELP subid_ldap_subid_total Total number of subid entries
	# TYPE subid_ldap_subid_total gauge
	subid_ldap_subid_total 4
	`

	if err := testutil.GatherAndCompare(metrics.MetricGathers(), strings.NewReader(expected),
		"subid_ldap_error", "subid_ldap_subid_added", "subid_ldap_subid_removed", "subid_ldap_subid_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}

	args = append([]string{
		fmt.Sprintf("--subid.subuid=%s", subuid),
		fmt.Sprintf("--subid.subgid=%s", subgid),
		fmt.Sprintf("--ldap.user-filter=%s", test.UserFilterStatus),
	}, baseArgs...)
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		t.Fatal(err)
	}
	err = run(logger)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subuidContent, err = os.ReadFile(subuid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	subgidContent, err = os.ReadFile(subgid)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedSubUID = `# File managed by subid-ldap
1000:65537:65536
1001:131074:65536
1002:262148:65536`
	if string(subuidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subuidContent), expectedSubUID)
	}
	if string(subgidContent) != expectedSubUID {
		t.Errorf("Unexpected subuid content:\nGot:\n%s\nExpected:\n%s", string(subgidContent), expectedSubUID)
	}

	expected = `
	# HELP subid_ldap_error Indicates an error was encountered
	# TYPE subid_ldap_error gauge
	subid_ldap_error 0
	# HELP subid_ldap_subid_added Number of subid entries added
	# TYPE subid_ldap_subid_added gauge
	subid_ldap_subid_added 0
	# HELP subid_ldap_subid_removed Number of subid entries removed
	# TYPE subid_ldap_subid_removed gauge
	subid_ldap_subid_removed 1
	# HELP subid_ldap_subid_total Total number of subid entries
	# TYPE subid_ldap_subid_total gauge
	subid_ldap_subid_total 3
	`

	if err := testutil.GatherAndCompare(metrics.MetricGathers(), strings.NewReader(expected),
		"subid_ldap_error", "subid_ldap_subid_added", "subid_ldap_subid_removed", "subid_ldap_subid_total"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}