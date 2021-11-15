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
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/lor00x/goldap/message"
	"github.com/treydock/subid-ldap/internal/utils"
	ldap "github.com/vjeantet/ldapserver"
)

const (
	BindDN           = "cn=test,dc=test"
	UserBaseDN       = "ou=People,dc=test"
	UserFilter       = "(objectClass=posixAccount)"
	UserFilterStatus = "(&(objectClass=posixAccount)(status=ACTIVE))"
	UserUIDAttr      = "uidNumber"
)

// GENCERTS: openssl req -newkey rsa:2048 -x509 -sha256 -days 3650 -nodes -out test.out -keyout test.key -subj "/C=US/ST=Ohio/L=Columbus/O=OSC/OU=OSC/CN=127.0.0.1"

// LocalhostCert is a PEM-encoded TLS cert with SAN DNS names
// "127.0.0.1" and "[::1]", expiring at the last second of 2049 (the end
// of ASN.1 time).
var LocalhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDkTCCAnmgAwIBAgIJAJHZ3TQ2IhkGMA0GCSqGSIb3DQEBCwUAMF8xCzAJBgNV
BAYTAlVTMQ0wCwYDVQQIDARPaGlvMREwDwYDVQQHDAhDb2x1bWJ1czEMMAoGA1UE
CgwDT1NDMQwwCgYDVQQLDANPU0MxEjAQBgNVBAMMCTEyNy4wLjAuMTAeFw0yMTAz
MDIxNzExNTJaFw0zMTAyMjgxNzExNTJaMF8xCzAJBgNVBAYTAlVTMQ0wCwYDVQQI
DARPaGlvMREwDwYDVQQHDAhDb2x1bWJ1czEMMAoGA1UECgwDT1NDMQwwCgYDVQQL
DANPU0MxEjAQBgNVBAMMCTEyNy4wLjAuMTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAMUNixh26lbTyMfNj4qz+28eUUeSEMfCsvp+nVqoab5gJ2N33NiR
EBhg6SU0Gw3TFbpEzg0UP9Fd9JJOEP0u1qOYawdisj9OZBS4AKEIo5Fuqko8Kzsi
0htfNabY9YdZPhtA1g0YTP3kpWKZ8HW+pe5xUlKfayFlw6WOiArEKQZGUhPlMl3w
QAB8M5PvraemboZFkDgcSQakbjHZZZbhRDrkzpIAWXzMdxJ35Au+143uv9N3FtQs
cBPoujdSpPVRBDXcA0+Uqxj2PX+Etszhy8Nuiph0NG2+qzV+aOf4/wcDtdcdDKnA
W7Zi6zsrlUoMGfcvSYvbndXDmzytNCH9z1cCAwEAAaNQME4wHQYDVR0OBBYEFDHH
/NPJu9pZqKYNc1zXcBT1b1zRMB8GA1UdIwQYMBaAFDHH/NPJu9pZqKYNc1zXcBT1
b1zRMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAGYVsS+NOUruCV6T
6oK4iVqtBrpNCcnC4noyfOxS4Um+pBcCfusKIxuMjsrMy6GRGAwVvnwNq3n3BsCz
oVxA1242eJBM8clNW2KdHeql5x2ivioh39kVpjQLEiI9B8X0ECw2IujnCa2+XsP4
zJs8So9zqMpHHYv+7cGUSJ/33W/4xdxTJdP49Xh5I6KzCuKoa0JIIBJl9aoK+Kl/
dOfo0E/1rKwr6cniTwEoUU8K/Am5kCUHdi5U8uyizR8zKnAhV7bIR680GSFFgfzK
BVpYVeZfEpeyRqC34OqBBWYNfxw3AqvA1s1WPYduxyl3sRwucB2hVhF00QHvSIvl
UkZmZZc=
-----END CERTIFICATE-----
`)

// localhostKey is the private key for LocalhostCert.
var localhostKey = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDFDYsYdupW08jH
zY+Ks/tvHlFHkhDHwrL6fp1aqGm+YCdjd9zYkRAYYOklNBsN0xW6RM4NFD/RXfSS
ThD9LtajmGsHYrI/TmQUuAChCKORbqpKPCs7ItIbXzWm2PWHWT4bQNYNGEz95KVi
mfB1vqXucVJSn2shZcOljogKxCkGRlIT5TJd8EAAfDOT762npm6GRZA4HEkGpG4x
2WWW4UQ65M6SAFl8zHcSd+QLvteN7r/TdxbULHAT6Lo3UqT1UQQ13ANPlKsY9j1/
hLbM4cvDboqYdDRtvqs1fmjn+P8HA7XXHQypwFu2Yus7K5VKDBn3L0mL253Vw5s8
rTQh/c9XAgMBAAECggEBAKdDGLd6cO2ktUAMF3Sn05v9gwaaUI4PkTaZdN24KJIF
Mkn3O0nE0IGw+RWwReqVK1NCBhkKACWqd+gcRcVzFZQl02ugdibQVplTmo0WNSlE
Y13B9vwqUWgUiAkJDliGAvbFMSxKXUgB5fRtMLPxUQ21uSgS06+0nr6P3qAs45nD
Zttxe9/g0k3HHwkNsxHM9A/RtvqG/b+l9znh6Lj3tsjBm8nB2rcm/TkQErOTIwQh
5mQogOlMgikApd8xlIhqnVGbfI0TP4zRw3VofvGa2C6xjH3Mqq6IaQTZ0/nj0lvW
vW6GHUkTJGMe2ketjUiYeIStPSSPQw7bUqAl0qHTDHkCgYEA77SZyS78vR4ypmND
LLmAuMWQ7oNI+5wrouL8cvMOT8mt6QM77HtT3W7PsQ8a9Ypt/xzOt9F7mn8kGQYF
64TFGBwSNHBaJ1nxFVUIjG2EV/4Bb3f21PuJLEmZnwfgkEcA/1GQUot0k9LKbbtN
eK/8ibFthSku8H3gqVCaC/pg7nMCgYEA0nKx1r+u0U/HXAzRImLdmA7BT5WDnBi1
/uHnI2XWvSADBqhXfwuUCPkkD5J4WJSwbkr8tcta+pUxPXS6wLqlXOW0aJBvfAKR
+pZ5StRRsgAcbvs+5ky9X8Bg8cjUPrBuwpde1BFTavmjQvAKRG0VMh8HPjs3QHTy
I2i02vtzHo0CgYEAhbhqUiE0PQwrlUaqoriZZnpQb74taK+maCfYTQfqY/hOXD7B
nxrtngnDMzMKBxBCbJ7VcxYZrgZfTNZfVxOqH9kJDtfeczVpmEznh+9QdQXuJxD1
UbtAusQUPvNWAyaZF9WYfXPuhMiCxNRIU5tZdjbUsgRXezG9sraUOTpj+KECgYBI
RrPlOTflEy044/3/fUz1qDukBYmJ1sLKovMrKRKzKYdghfhm3acd3dMQthE2+voN
Jxvbo9e/L/YVUT3Ca1fXq9xl/RUM1iUklwFZPcpBA+DADPHxTnHLrNqer4aVcSrZ
Efuzga/QkaQMnTwpe/1HlXh7WwMC1CdFGfTjMHC9EQKBgQDAfL1d5E/1JDX0U3lf
hda2KPTtGj2ZfouHiUdmB4Ma8drXldVFP2atNQMZ3jNdzdQQn/1jX54i/qySRvYZ
l48gnDophKhiriOd4Zph3vbl3eZE0+IOXLADgI51tMmxL0gybEPrlEtVaItNd48G
mfc9r598/by35iVqsCWBf2o3/Q==
-----END PRIVATE KEY-----

`)

func LdapServer() *ldap.Server {
	//ldap.Logger = log.New(os.Stdout, "[server] ", log.LstdFlags)
	ldap.Logger = log.New(io.Discard, "[server] ", log.LstdFlags)
	server := ldap.NewServer()
	routes := ldap.NewRouteMux()
	//routes.NotFound(handleNotFound)
	routes.Bind(handleBind)
	routes.Search(handleSearchUser).
		BaseDn(UserBaseDN).
		Filter(UserFilter).
		Label("SEARCH - USER")
	routes.Search(handleSearchUser).
		BaseDn(UserBaseDN).
		Filter(UserFilterStatus).
		Label("SEARCH - USER")
	//routes.Search(handleSearch).Label("SEARCH - NO MATCH")
	routes.Extended(handleStartTLS).RequestName(ldap.NoticeOfStartTLS).Label("StartTLS")
	server.Handle(routes)
	return server
}

func handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	if r.AuthenticationChoice() == "simple" {
		if string(r.Name()) != BindDN {
			res.SetResultCode(ldap.LDAPResultInvalidCredentials)
			res.SetDiagnosticMessage("invalid credentials")
		}
	}
	w.Write(res)
}

/*func handleNotFound(w ldap.ResponseWriter, r *ldap.Message) {
	switch r.ProtocolOpType() {
	case ldap.ApplicationBindRequest:
		res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
		res.SetDiagnosticMessage("Default binding behavior set to return Success")

		w.Write(res)

	default:
		res := ldap.NewResponse(ldap.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Operation not implemented by server")
		w.Write(res)
	}
}*/

func handleSearchUser(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	data := map[string]map[string][]string{
		"testuser1": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1000"},
			"status":      []string{"ACTIVE"},
		},
		"testuser2": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1001"},
			"status":      []string{"ACTIVE"},
		},
		"testuser3": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1002"},
			"status":      []string{"ACTIVE"},
		},
		"testuser4": {
			"objectClass": []string{"posixAccount"},
			"uidNumber":   []string{"1003"},
			"status":      []string{"RESTRICTED"},
		},
	}
	for cn, attrs := range data {
		dn := fmt.Sprintf("cn=%s,%s", cn, r.BaseObject())
		e := ldap.NewSearchResultEntry(dn)
		e.AddAttribute("cn", message.AttributeValue(cn))
		e.AddAttribute("uid", message.AttributeValue(cn))
		if strings.Contains(r.FilterString(), "status=ACTIVE") {
			if !utils.SliceContains(attrs["status"], "ACTIVE") {
				continue
			}
		}
		for key, value := range attrs {
			values := []message.AttributeValue{}
			for _, v := range value {
				values = append(values, message.AttributeValue(v))
			}
			e.AddAttribute(message.AttributeDescription(key), values...)
		}
		w.Write(e)
	}
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

/*func handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultNoSuchObject)
	w.Write(res)
}*/

func handleStartTLS(w ldap.ResponseWriter, m *ldap.Message) {
	res := ldap.NewExtendedResponse(ldap.LDAPResultSuccess)
	cert, err := tls.X509KeyPair(LocalhostCert, localhostKey)
	if err != nil {
		log.Printf("StartTLS cert parse error %v", err)
		res.SetDiagnosticMessage(fmt.Sprintf("StartTLS cert parse error : \"%s\"", err.Error()))
		res.SetResultCode(ldap.LDAPResultOperationsError)
	}

	tlsconfig := &tls.Config{
		MinVersion:   tls.VersionTLS10,
		MaxVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ServerName:   "127.0.0.1",
	}
	tlsConn := tls.Server(m.Client.GetConn(), tlsconfig)
	res.SetResponseName(ldap.NoticeOfStartTLS)
	w.Write(res)

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("StartTLS Handshake error %v", err)
		res.SetDiagnosticMessage(fmt.Sprintf("StartTLS Handshake error : \"%s\"", err.Error()))
		res.SetResultCode(ldap.LDAPResultOperationsError)
		w.Write(res)
		return
	}

	m.Client.SetConn(tlsConn)
	log.Println("StartTLS OK")
}
