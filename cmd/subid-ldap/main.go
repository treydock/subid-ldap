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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/treydock/subid-ldap/internal/config"
	localldap "github.com/treydock/subid-ldap/internal/ldap"
	"github.com/treydock/subid-ldap/internal/metrics"
	"github.com/treydock/subid-ldap/internal/subid"
	"github.com/treydock/subid-ldap/internal/utils"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	subUIDPath           = kingpin.Flag("subid.subuid", "Path to subuid file").Default(subid.SubUIDPath).String()
	subGIDPath           = kingpin.Flag("subid.subgid", "Path to subgid file").Default(subid.SubGIDPath).String()
	subIDStart           = kingpin.Flag("subid.start", "Start ID of subuid/subgid").Default("65537").Int()
	subIDRange           = kingpin.Flag("subid.range", "Range for each entry").Default("65536").Int()
	ldapURL              = kingpin.Flag("ldap.url", "LDAP URL").Required().String()
	ldapTLS              = kingpin.Flag("ldap.tls", "Enable TLS connection to LDAP server").Default("false").Bool()
	ldapTLSVerify        = kingpin.Flag("ldap.tls-verify", "Verify TLS certificate with LDAP server").Default("true").Bool()
	ldapTLSCACert        = kingpin.Flag("ldap.tls-ca-cert", "TLS CA Cert for LDAP server").String()
	ldapUserBaseDN       = kingpin.Flag("ldap.user-base-dn", "LDAP User Base DN").Required().String()
	ldapUserFilter       = kingpin.Flag("ldap.user-filter", "LDAP user filter").Default("(objectClass=posixAccount)").String()
	ldapUserUIDAttr      = kingpin.Flag("ldap.user-uid-attr", "LDAP user UID attribute").Default("uidNumber").String()
	ldapBindDN           = kingpin.Flag("ldap.bind-dn", "LDAP Bind DN").String()
	ldapBindPassword     = kingpin.Flag("ldap.bind-password", "LDAP Bind Password").String()
	ldapPagedSearch      = kingpin.Flag("ldap.paged-search", "Enable LDAP paged searching").Default("false").Bool()
	ldapPagedSearchSize  = kingpin.Flag("ldap.paged-search-size", "LDAP paged search size").Default("1000").Int()
	daemon               = kingpin.Flag("daemon", "Run application as a daemon").Default("false").Bool()
	daemonUpdateInterval = kingpin.Flag("daemon.update-interval", "How often to update in daemon mode").Default("5m").Duration()
	listenAddress        = kingpin.Flag("metrics.listen-address", "Address to listen on for daemon metrics").Default(":8085").String()
	metricsPath          = kingpin.Flag("metrics.path", "Path to save Prometheus metrics when not daemon").Default("").String()
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

	metricGathers := metrics.MetricGathers()
	if *daemon {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
		             <head><title>` + config.AppName + `</title></head>
		             <body>
		             <h1>` + config.AppName + `</h1>
		             <p><a href='` + config.MetricsPath + `'>Metrics</a></p>
		             </body>
		             </html>`))
		})
		http.Handle(config.MetricsPath, promhttp.HandlerFor(metricGathers, promhttp.HandlerOpts{}))
		go func() {
			if err := http.ListenAndServe(*listenAddress, nil); err != nil {
				level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
				os.Exit(1)
			}
		}()
	}

	for {
		var exitCode int
		start := time.Now()
		metrics.MetricLastRun.Set(float64(start.Unix()))
		err = run(logger)
		metrics.MetricDuration.Set(time.Since(start).Seconds())
		if err != nil {
			metrics.MetricError.Set(1)
			level.Error(logger).Log("err", err)
		}
		if *daemon {
			time.Sleep(*daemonUpdateInterval)
		} else {
			if *metricsPath != "" {
				err = metrics.MetricsWrite(*metricsPath, metricGathers)
				if err != nil {
					level.Error(logger).Log("msg", "Failed to write metrics file", "err", err)
				}
			}
			os.Exit(exitCode)
		}
	}
}

func run(logger log.Logger) error {
	config := &config.Config{
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
	l, err := localldap.LDAPConnect(config, logger)
	if err != nil {
		return err
	}
	defer l.Close()
	users, err := localldap.LDAPUsers(l, config, logger)
	if err != nil {
		return err
	}
	utils.SortSliceStringInts(&users)
	subid.SubUIDPath = *subUIDPath
	subid.SubGIDPath = *subGIDPath
	level.Debug(logger).Log("msg", "LDAP returned users count", "count", len(users))
	runLogger := log.With(logger, "subuid", subid.SubUIDPath)
	managed, err := subid.SubIDManaged(subid.SubUIDPath, runLogger)
	if err != nil {
		level.Error(runLogger).Log("msg", "Failed to check managed state of subid", "err", err)
	}
	if managed {
		subids := subid.SubIDGenerate(config, logger)
		existingSubIDs, err := subid.SubIDLoad(subid.SubUIDPath, runLogger)
		if err != nil {
			level.Error(runLogger).Log("msg", "Failed to load subid file", "err", err)
			return err
		}
		level.Debug(runLogger).Log("msg", "Existing subuids loaded", "count", len(*existingSubIDs))
		err = subid.SubIDUpdate(users, existingSubIDs, subids, subid.SubUIDPath, runLogger)
		if err != nil {
			level.Error(runLogger).Log("msg", "Failed to update subid file", "err", err)
			return err
		}
		level.Info(runLogger).Log("msg", "Successfully updated subids")
	} else {
		err = subid.SubIDSaveNew(users, subid.SubUIDPath, config)
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
