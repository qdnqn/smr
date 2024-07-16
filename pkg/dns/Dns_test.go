package dns

import (
	"errors"
	"fmt"
	"github.com/go-playground/assert/v2"
	"github.com/miekg/dns"
	"github.com/simplecontainer/smr/pkg/static"
	"net"
	"strings"
	"testing"
)

func TestAddARecord(t *testing.T) {
	type Wanted struct {
		response []string
	}

	type Parameters struct {
		domain string
		ip     string
	}

	testCases := []struct {
		name       string
		mockFunc   func()
		wanted     Wanted
		parameters Parameters
	}{
		{
			"Valid format",
			func() {
			},
			Wanted{
				response: []string{"10.0.0.2"},
			},
			Parameters{
				domain: fmt.Sprintf("mysql.mysql-mysql-1.%s", static.SMR_LOCAL_DOMAIN),
				ip:     "10.0.0.2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockFunc()

			dnsRecords := New()

			dnsRecords.AddARecord(tc.parameters.domain, tc.parameters.ip)
			response := dnsRecords.Find(tc.parameters.domain)

			assert.Equal(t, tc.wanted.response, response)
		})
	}
}

func TestRemoveARecord(t *testing.T) {
	type Wanted struct {
		error error
	}

	type Parameters struct {
		domain string
		ip     string
		ip2    string
	}

	testCases := []struct {
		name       string
		mockFunc   func()
		wanted     Wanted
		parameters Parameters
	}{
		{
			"Valid delete",
			func() {
			},
			Wanted{
				error: nil,
			},
			Parameters{
				domain: fmt.Sprintf("mysql.mysql-mysql-1.%s", static.SMR_LOCAL_DOMAIN),
				ip:     "10.0.0.2",
				ip2:    "10.0.0.3",
			},
		},
		{
			"Invalid delete",
			func() {
			},
			Wanted{
				error: errors.New(fmt.Sprintf("ip 10.0.0.2 not found for specifed domain mysql.mysql-mysql-1.%s", static.SMR_LOCAL_DOMAIN)),
			},
			Parameters{
				domain: fmt.Sprintf("mysql.mysql-mysql-1.%s", static.SMR_LOCAL_DOMAIN),
				ip:     "10.0.0.2",
				ip2:    "10.0.0.3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockFunc()

			dnsRecords := New()

			if tc.name == "Valid delete" {
				dnsRecords.AddARecord(tc.parameters.domain, tc.parameters.ip)
			}

			err := dnsRecords.RemoveARecord(tc.parameters.domain, tc.parameters.ip)

			assert.Equal(t, tc.wanted.error, err)
		})
	}
}

func TestParseQuery(t *testing.T) {
	type Wanted struct {
		response string
		error    error
	}

	type Parameters struct {
		cache  *Records
		msg    *dns.Msg
		domain string
		ip     string
	}

	testCases := []struct {
		name       string
		mockFunc   func()
		wanted     Wanted
		parameters Parameters
	}{
		{
			"Valid domain inside",
			func() {
			},
			Wanted{
				response: "10.0.0.2",
				error:    nil,
			},
			Parameters{
				cache:  &Records{},
				msg:    &dns.Msg{},
				domain: fmt.Sprintf("mysql.mysql-mysql-1.%s", static.SMR_LOCAL_DOMAIN),
				ip:     "10.0.0.2",
			},
		},
		{
			"Valid domain outside",
			func() {
			},
			Wanted{
				response: "",
				error:    nil,
			},
			Parameters{
				cache:  &Records{},
				msg:    &dns.Msg{},
				domain: "google.com",
				ip:     "10.0.0.2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockFunc()

			dnsRecords := New()

			if tc.name == "Valid domain inside" {
				dnsRecords.AddARecord(tc.parameters.domain, tc.parameters.ip)
			}

			m := new(dns.Msg)

			q := new(dns.Msg)
			q.SetQuestion(tc.parameters.domain, dns.TypeA)

			m.SetReply(q)
			m.Compress = false

			m.SetReply(q)
			m.Compress = false

			err := ParseQuery(dnsRecords, m)

			for _, answer := range m.Answer {
				split := strings.Split(answer.String(), "\t")
				ip := split[len(split)-1]
				netIp := net.ParseIP(ip)

				if netIp.IsPrivate() {
					assert.Equal(t, tc.wanted.response, netIp.String())
				} else {
					assert.Equal(t, false, netIp.IsPrivate())
				}
			}

			assert.Equal(t, tc.wanted.error, err)
		})
	}
}
