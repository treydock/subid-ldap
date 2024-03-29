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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/treydock/subid-ldap/internal/config"
	localldap "github.com/treydock/subid-ldap/internal/ldap"
	"github.com/treydock/subid-ldap/internal/metrics"
	"github.com/treydock/subid-ldap/internal/subid"
	"github.com/treydock/subid-ldap/internal/utils"
)

var (
	subUIDPath           = kingpin.Flag("subid.subuid", "Path to subuid file").Default(subid.SubUIDPath).Envar("SUBID_SUBUID").String()
	subGIDPath           = kingpin.Flag("subid.subgid", "Path to subgid file").Default(subid.SubGIDPath).Envar("SUBID_SUBGID").String()
	subIDStart           = kingpin.Flag("subid.start", "Start ID of subuid/subgid").Default("65537").Envar("SUBID_START").Int()
	subIDRange           = kingpin.Flag("subid.range", "Range for each entry").Default("65536").Envar("SUBID_RANGE").Int()
	ldapURL              = kingpin.Flag("ldap.url", "LDAP URL").Required().Envar("LDAP_URL").String()
	ldapTLS              = kingpin.Flag("ldap.tls", "Enable TLS connection to LDAP server").Default("false").Envar("LDAP_TLS").Bool()
	ldapTLSVerify        = kingpin.Flag("ldap.tls-verify", "Verify TLS certificate with LDAP server").Default("true").Envar("LDAP_TLS_VERIFY").Bool()
	ldapTLSCACert        = kingpin.Flag("ldap.tls-ca-cert", "TLS CA Cert for LDAP server").Envar("LDAP_TLS_CA_CERT").String()
	ldapUserBaseDN       = kingpin.Flag("ldap.user-base-dn", "LDAP User Base DN").Required().Envar("LDAP_USER_BASE_DN").String()
	ldapUserFilter       = kingpin.Flag("ldap.user-filter", "LDAP user filter").Default("(objectClass=posixAccount)").Envar("LDAP_USER_FILTER").String()
	ldapUserUIDAttr      = kingpin.Flag("ldap.user-uid-attr", "LDAP user UID attribute").Default("uidNumber").Envar("LDAP_USER_UID_ATTR").String()
	ldapBindDN           = kingpin.Flag("ldap.bind-dn", "LDAP Bind DN").Envar("LDAP_BIND_DN").String()
	ldapBindPassword     = kingpin.Flag("ldap.bind-password", "LDAP Bind Password").Envar("LDAP_BIND_PASSWORD").String()
	ldapPagedSearch      = kingpin.Flag("ldap.paged-search", "Enable LDAP paged searching").Default("false").Envar("LDAP_PAGED_SEARCH").Bool()
	ldapPagedSearchSize  = kingpin.Flag("ldap.paged-search-size", "LDAP paged search size").Default("1000").Envar("LDAP_PAGED_SEARCH_SIZE").Int()
	daemon               = kingpin.Flag("daemon", "Run application as a daemon").Default("false").Envar("DAEMON").Bool()
	daemonUpdateInterval = kingpin.Flag("daemon.update-interval", "How often to update in daemon mode").Default("5m").Envar("DAEMON_UPDATE_INTERVAL").Duration()
	listenAddress        = kingpin.Flag("metrics.listen-address", "Address to listen on for daemon metrics").Default(":8085").Envar("METRICS_LISTEN_ADDRESS").String()
	metricsPath          = kingpin.Flag("metrics.path", "Path to save Prometheus metrics when not daemon").Default("").Envar("METRICS_PATH").String()
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print(config.AppName))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	err := validateArgs(logger)
	if err != nil {
		os.Exit(1)
	}

	level.Info(logger).Log("msg", fmt.Sprintf("Starting %s", config.AppName), "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	if *daemon {
		go func() {
			if err := metrics.MetricsServer(*listenAddress); err != nil {
				level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
				os.Exit(1)
			}
		}()
	}

	for {
		var exitCode int
		err = run(logger)
		if err != nil {
			level.Error(logger).Log("err", err)
		}
		if *daemon {
			time.Sleep(*daemonUpdateInterval)
		} else {
			os.Exit(exitCode)
		}
	}
}

func run(logger log.Logger) error {
	var err error
	metrics.MetricLastRun.Set(float64(time.Now().Unix()))
	if !*daemon && *metricsPath != "" {
		defer metrics.MetricsWrite(*metricsPath, metrics.MetricGathers(false), logger)
	}
	defer metrics.Duration()()
	defer metrics.Error()(&err)
	c := &config.Config{
		LdapURL:         *ldapURL,
		LdapTLS:         *ldapTLS,
		LdapTLSVerify:   *ldapTLSVerify,
		LdapTLSCACert:   *ldapTLSCACert,
		BindDN:          *ldapBindDN,
		BindPassword:    *ldapBindPassword,
		UserBaseDN:      *ldapUserBaseDN,
		UserFilter:      *ldapUserFilter,
		UserUIDAttr:     *ldapUserUIDAttr,
		PagedSearch:     *ldapPagedSearch,
		PagedSearchSize: *ldapPagedSearchSize,
		SubIDStart:      *subIDStart,
		SubIDRange:      *subIDRange,
	}
	l, err := localldap.LDAPConnect(c, logger)
	if err != nil {
		return err
	}
	defer l.Close()
	users, err := localldap.LDAPUsers(l, c, logger)
	if err != nil {
		return err
	}
	utils.SortSliceStringInts(&users)
	subid.SubUIDPath = *subUIDPath
	subid.SubGIDPath = *subGIDPath
	level.Debug(logger).Log("msg", "LDAP returned users count", "count", len(users))
	runLogger := log.With(logger, "subuid", subid.SubUIDPath)
	managed, err := subid.SubIDManaged(subid.SubUIDPath, c, runLogger)
	if err != nil {
		level.Error(runLogger).Log("msg", "Failed to check managed state of subid", "err", err)
	}
	if managed {
		subids := subid.SubIDGenerate(c, logger)
		existingSubIDs, err := subid.SubIDLoad(subid.SubUIDPath, runLogger)
		if err != nil {
			level.Error(runLogger).Log("msg", "Failed to load subid file", "err", err)
			return err
		}
		level.Debug(runLogger).Log("msg", "Existing subuids loaded", "count", len(*existingSubIDs))
		err = subid.SubIDUpdate(users, existingSubIDs, subids, subid.SubUIDPath, c, runLogger)
		if err != nil {
			level.Error(runLogger).Log("msg", "Failed to update subid file", "err", err)
			return err
		}
		level.Info(runLogger).Log("msg", "Successfully updated subids")
	} else {
		err = subid.SubIDSaveNew(users, subid.SubUIDPath, c)
		if err != nil {
			level.Error(runLogger).Log("msg", "Failed to save new subid file", "err", err)
			return err
		}
		level.Info(runLogger).Log("msg", "Successfully create subids")
	}
	err = subid.SubGIDSave(subid.SubUIDPath, subid.SubGIDPath)
	if err != nil {
		level.Error(runLogger).Log("msg", "Failed to copy subuid to subgid", "subgid", subid.SubGIDPath, "err", err)
		return err
	}

	return nil
}

func validateArgs(logger log.Logger) error {
	errs := []string{}
	var err error
	if (*ldapBindDN != "" && *ldapBindPassword == "") || (*ldapBindDN == "" && *ldapBindPassword != "") {
		errs = append(errs, "ldap-bind=\"Must provide both LDAP Bind DN and Bind Password if either is provided\"")
	}
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, ", "))
		level.Error(logger).Log("err", err)
	}
	return err
}
