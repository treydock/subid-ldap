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

package config

const (
	AppName = "subid-ldap"
)

type Config struct {
	LdapURL         string
	LdapTLS         bool
	LdapTLSVerify   bool
	LdapTLSCACert   string
	BindDN          string
	BindPassword    string
	UserBaseDN      string
	UserFilter      string
	UserUIDAttr     string
	PagedSearch     bool
	PagedSearchSize int
	SubIDStart      int
	SubIDRange      int
}
