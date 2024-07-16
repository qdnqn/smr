package dns

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"net"
)

func New() *Records {
	return &Records{}
}

func (r *Records) AddARecord(domain string, ip string) {
	domain = fmt.Sprintf("%s.", domain)

	if len(r.ARecords) > 0 {
		_, ARecordexists := r.ARecords[domain]

		if !ARecordexists {
			tmp := r.ARecords
			tmp[domain] = ARecord{
				map[string][]string{
					domain: {ip},
				},
				map[string][]string{
					domain: {},
				},
			}

			r.ARecords = tmp
		}

		_, domainexists := r.ARecords[domain].Domain[domain]

		if domainexists {
			var contains bool
			for _, i := range r.ARecords[domain].Domain[domain] {
				if i == ip {
					contains = true
				}
			}

			if !contains {
				r.ARecords[domain].Domain[domain] = append(r.ARecords[domain].Domain[domain], ip)
			}
		} else {
			tmp := ARecord{
				map[string][]string{
					domain: {ip},
				},
				map[string][]string{
					domain: {},
				},
			}

			r.ARecords[domain] = tmp
		}
	} else {
		tmp := map[string]ARecord{
			domain: {
				map[string][]string{
					domain: {ip},
				},
				map[string][]string{
					domain: {},
				},
			},
		}

		r.ARecords = tmp
	}
}
func (r *Records) RemoveARecord(domain string, ip string) error {
	ips := r.Find(domain)

	if len(ips) > 0 {
		for i, v := range ips {
			if v == ip {
				ips = append(ips[:i], ips[i+1:]...)
			}
		}

		r.ARecords[domain].Domain[domain] = ips

		return nil
	} else {
		return errors.New(fmt.Sprintf("ip %s not found for specifed domain %s", ip, domain))
	}
}

func (r *Records) RemoveARecordQueue(domain string, ip string) error {
	ips := r.Find(domain)

	if len(ips) > 0 {
		for _, v := range ips {
			if v == ip {
				r.ARecords[domain].DomainDelete[domain] = append(r.ARecords[domain].DomainDelete[domain], v)
			}
		}

		return nil
	} else {
		return errors.New(fmt.Sprintf("ip %s not found for specifed domain %s", ip, domain))
	}
}

func (r *Records) Find(domain string) []string {
	arecords, exists := r.ARecords[domain]

	if exists {
		return arecords.Domain[domain]
	} else {
		return []string{}
	}
}

func (r *Records) FindDeleteQueue(domain string) []string {
	arecords, exists := r.ARecords[domain]

	if exists {
		return arecords.DomainDelete[domain]
	} else {
		return []string{}
	}
}

func (r *Records) ResetDeleteQueue(domain string) {
	Arecords, exists := r.ARecords[domain]

	if exists {
		Arecords.DomainDelete[domain] = nil
	}
}

func ParseQuery(cache *Records, m *dns.Msg) error {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			ips := cache.Find(q.Name)

			if len(ips) > 0 {
				var rr dns.RR
				var err error

				for _, ip := range ips {
					rr, err = dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))

					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
			} else {
				config := dns.ClientConfig{
					Servers:  []string{"8.8.8.8"},
					Port:     "53",
					Ndots:    0,
					Timeout:  1,
					Attempts: 5,
				}

				c := new(dns.Client)

				ms := new(dns.Msg)
				ms.SetQuestion(dns.Fqdn(q.Name), q.Qtype)
				ms.RecursionDesired = true

				r, _, err := c.Exchange(ms, net.JoinHostPort(config.Servers[0], config.Port))

				if err != nil {
					return err
				}

				if r == nil {
					return errors.New("response empty")
				}

				if r.Rcode != dns.RcodeSuccess {
					return errors.New("request failed")
				}

				for _, a := range r.Answer {
					m.Answer = append(m.Answer, a)
				}
			}
		}
	}

	return nil
}
