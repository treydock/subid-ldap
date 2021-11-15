// Copyright 2021 Trey Dockendof
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

package ldap

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/treydock/subid-ldap/internal/config"
	"github.com/treydock/subid-ldap/internal/test"
)

const (
	ldapserver = "127.0.0.1:10391"
)

func getConfig() *config.Config {
	return &config.Config{
		LdapURL:     fmt.Sprintf("ldap://%s", ldapserver),
		BindDN:      test.BindDN,
		UserBaseDN:  test.UserBaseDN,
		UserFilter:  test.UserFilter,
		UserUIDAttr: test.UserUIDAttr,
	}
}

func TestMain(m *testing.M) {
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

func TestLDAPConnectErr(t *testing.T) {
	_config := getConfig()
	_config.LdapURL = "ldap://dne:389"
	_, err := LDAPConnect(_config, log.NewNopLogger())
	if err == nil {
		t.Errorf("Expected an error with invalid LdapURL")
	}
}

func TestLDAPConnectBind(t *testing.T) {
	_config := getConfig()
	_config.BindPassword = "test"
	_, err := LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Errorf("Unexpected error during BIND: %s", err.Error())
	}
}

func TestLDAPConnectBindInvalid(t *testing.T) {
	_config := getConfig()
	_config.BindDN = "cn=foobar"
	_config.BindPassword = "test"
	_, err := LDAPConnect(_config, log.NewNopLogger())
	if err == nil {
		t.Errorf("Expected an error with invalid BIND")
	}
}

func TestLDAPConnectTLS(t *testing.T) {
	_config := getConfig()
	_config.LdapTLS = true
	_config.LdapTLSCACert = string(test.LocalhostCert)
	_, err := LDAPConnect(_config, log.NewNopLogger())
	if err != nil {
		t.Errorf("Unexpected error during StartTLS: %s", err.Error())
	}
}

func TestLDAPConnectTLSError(t *testing.T) {
	_config := getConfig()
	_config.LdapURL = "ldap://localhost:389"
	_config.LdapTLS = true
	_config.LdapTLSCACert = string(test.LocalhostCert)
	_, err := LDAPConnect(_config, log.NewNopLogger())
	if err == nil {
		t.Errorf("Expected an error with invalid TLS ServerName")
	}
}
