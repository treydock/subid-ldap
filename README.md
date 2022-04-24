[![CI Status](https://github.com/treydock/subid-ldap/actions/workflows/test.yaml/badge.svg?branch=main)](https://github.com/treydock/subid-ldap/actions?query=workflow%3Atest)
[![GitHub release](https://img.shields.io/github/v/release/treydock/subid-ldap?include_prereleases&sort=semver)](https://github.com/treydock/subid-ldap/releases/latest)
![GitHub All Releases](https://img.shields.io/github/downloads/treydock/subid-ldap/total)
![Docker Pulls](https://img.shields.io/docker/pulls/treydock/subid-ldap)
[![Go Report Card](https://goreportcard.com/badge/github.com/treydock/subid-ldap?ts=1)](https://goreportcard.com/report/github.com/treydock/subid-ldap)
[![codecov](https://codecov.io/gh/treydock/subid-ldap/branch/main/graph/badge.svg)](https://codecov.io/gh/treydock/subid-ldap)

# subid-ldap

The subid-ldap tool is intended to generate `/etc/subuid` and `/etc/subgid` based on LDAP data.

The entries in `/etc/subuid` and `/etc/subgid` are merged with new data so that existing entries keep
their designated ID when new entries are added or old entries are removed.

The LDAP user UID is used by default for improved performance with tools using the subuid/subgid entries.

The contents of `/etc/subuid` are copied to `/etc/subgid` when changes are made.

## Install

### Install from archive

```
wget -O /tmp/subid-ldap.tar.gz https://github.com/treydock/subid-ldap/releases/download/v0.2.0/subid-ldap_0.2.0_linux_amd64.tar.gz
mkdir /usr/local/share/subid-ldap
tar xf /tmp/subid-ldap.tar.gz -C /usr/local/share/subid-ldap
ln -s /usr/local/share/subid-ldap/subid-ldap /usr/local/sbin/subid-ldap
```

If running subid-ldap as a daemon, install the systemd unit file:

```
cp /usr/local/share/subid-ldap/subid-ldap.service /etc/systemd/system/subid-ldap.service
```

The environment file `/etc/sysconfig/subid-ldap` would need to contain necessary configurations or directly edit
`/etc/systemd/system/subid-ldap.service` to add the necessary flag.

### Docker

Add additional flags either via additional environment variables or passing the flags after the image name.

```
docker run --detach --rm --name subid-ldap \
  -v /etc/subuid:/host/subuid -v /etc/subgid:/host/subgid \
  -e SUBID_SUBUID=/host/subuid -e SUBID_SUBGID=/host/subgid \
  -e LDAP_URL=ldap://example.com -e DAEMON=true treydock/subid-ldap:latest
```

## Configuration

The subid-ldap can be run as daemon with `--daemon` flag or executed via cron.

For Active Directory it's likely paged searches are required so at minimum the `--ldap-paged-search` flag would be required.

The following flags and environment variables can modify the behavior of the subid-ldap:

| Flag    | Environment Variable | Description | Default/Required |
|---------|----------------------|-------------|------------------|
| --subid.subuid | SUBID_SUBUID | Path to subuid file | `/etc/subuid` |
| --subid.subgid | SUBID_SUBGID | Path to subgid file | `/etc/subgid` |
| --subid.start | SUBID_START | Start ID of subuid/subgid | `65537` |
| --subid.range | SUBID_RANGE | Range for each entry | `65536` |
| --ldap.url | LDAP_URL | LDAP URL to query, example: `ldap://ldap.example.com:389` | **Required** |
| --ldap.tls | LDAP_TLS | Enable TLS when connecting to LDAP | `false` |
| --no-ldap.tls-verify | LDAP_TLS_VERIFY=false | Disable TLS verification when connecting to LDAP | `true` |
| --ldap.tls-ca-cert | LDAP_TLS_CA_CERT | The contents of TLS CA cert when the certificate needs to be verified and not in global trust store | None |
| --ldap.user-base-dn | LDAP_USER_BASE_DN | Base DN of the Users OU in LDAP | **Required** |
| --ldap.bind-dn | LDAP_BIND_DN | Bind DN when connecting to LDAP | None (anonymous binds) |
| --ldap.bind-password | LDAP_BIND_PASSWORD | Bind password when connecting to LDAP | None (anonymous binds) |
| --ldap.user-filter | LDAP_USER_FILTER | User LDAP filter | `(objectClass=posixAccount)` |
| --ldap.user-uid-attr | LDAP_USER_UID_ATTR | LDAP user UID attribute | `uidNumber` |
| --ldap.paged-search | LDAP_PAGED_SEARCH | Enable paged searches against LDAP | `false` |
| --ldap.paged-search-size | LDAP_PAGED_SEARCH_SIZE | Size of searches when using paged searches | `1000` |
| --daemon | DAEMON | Run as daemon | `false` |
| --daemon.update-interval | DAEMON_UPDATE_INTERVAL | Update interval in daemon mode | `5m` |
| --metrics.listen-address | METRICS_LISTEN_ADDRESS | The address to listen on for metrics when running as daemon | `:8085` |
| --metrics.path | METRICS_PATH | The path to store metrics that can be scraped by node_exporter | |
