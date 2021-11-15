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

package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/url"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/treydock/subid-ldap/internal/config"
)

func LDAPConnect(config *config.Config, logger log.Logger) (*ldap.Conn, error) {
	level.Debug(logger).Log("msg", "Connecting to LDAP", "url", config.LdapURL)
	l, err := ldap.DialURL(config.LdapURL)
	if err != nil {
		level.Error(logger).Log("msg", "Error connecting to LDAP URL", "url", config.LdapURL, "err", err)
		return l, err
	}
	if config.LdapTLS {
		err = LDAPTLS(l, config, logger)
		if err != nil {
			return l, err
		}
	}
	if config.BindDN != "" && config.BindPassword != "" {
		level.Debug(logger).Log("msg", "Binding to LDAP", "url", config.LdapURL, "binddn", config.BindDN)
		err = l.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			level.Error(logger).Log("msg", "Error binding to LDAP", "binddn", config.BindDN, "err", err)
			return l, err
		}
	}
	return l, err
}

func LDAPTLS(l *ldap.Conn, config *config.Config, logger log.Logger) error {
	var err error
	u, err := url.Parse(config.LdapURL)
	if err != nil {
		level.Error(logger).Log("msg", "Error parsing LDAP URL", "url", config.LdapURL, "err", err)
		return err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		level.Error(logger).Log("msg", "Error getting LDAP host name", "host", u.Host, "err", err)
		return err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !config.LdapTLSVerify,
		ServerName:         host,
	}
	if config.LdapTLSCACert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(config.LdapTLSCACert))
		tlsConfig.RootCAs = caCertPool
	}
	level.Debug(logger).Log("msg", "Performing Start TLS with LDAP server")
	err = l.StartTLS(tlsConfig)
	if err != nil {
		level.Error(logger).Log("msg", "Error starting TLS for LDAP connection", "err", err)
	}
	return err
}

func LDAPUsers(l *ldap.Conn, config *config.Config, logger log.Logger) ([]string, error) {
	users := []string{}
	attrs := []string{config.UserUIDAttr}
	level.Debug(logger).Log("msg", "Running user search", "basedn", config.UserBaseDN, "filter", config.UserFilter, "attr", config.UserUIDAttr)
	request := ldap.NewSearchRequest(config.UserBaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		config.UserFilter, attrs, nil)
	result, err := LDAPSearch(l, request, "user", config, logger)
	for _, entry := range result.Entries {
		uid := entry.GetAttributeValue(config.UserUIDAttr)
		users = append(users, uid)
	}
	return users, err
}

func LDAPSearch(l *ldap.Conn, request *ldap.SearchRequest, queryType string, config *config.Config, logger log.Logger) (*ldap.SearchResult, error) {
	var result *ldap.SearchResult
	var err error
	if config.PagedSearch {
		result, err = l.SearchWithPaging(request, uint32(config.PagedSearchSize))
	} else {
		result, err = l.Search(request)
	}
	if err != nil {
		level.Error(logger).Log("msg", "Error getting results", "type", queryType, "err", err)
	} else {
		level.Debug(logger).Log("msg", "results", "type", queryType, "count", len(result.Entries))
	}
	return result, err
}
