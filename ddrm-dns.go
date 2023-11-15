//go:build client
// +build client

package main

import (
	"os"

	"github.com/miekg/dns"
)

// Simple record type to help make the JSON parsing easier
type DdrmRecordType string

const (
	//lint:ignore U1000 Ignore unused types: they are used during JSON parsing but go-staticcheck gets upset because it can't know that
	ddrmRecordTypeA     DdrmRecordType = "A"
	ddrmRecordTypeAAAA  DdrmRecordType = "AAAA"
	ddrmRecordTypeTXT   DdrmRecordType = "TXT"
	ddrmRecordTypeMX    DdrmRecordType = "MX"
	ddrmRecordTypeCAA   DdrmRecordType = "CAA"
	ddrmRecordTypeCNAME DdrmRecordType = "CNAME"
	ddrmRecordTypeNS    DdrmRecordType = "NS"
	ddrmRecordTypePTR   DdrmRecordType = "PTR"
	ddrmRecordTypeSOA   DdrmRecordType = "SOA"
	ddrmRecordTypeSRV   DdrmRecordType = "SRV"
)

// try and ask a question to get a record's records
func getRecordData(fqdn string, recordtype DdrmRecordType) (answer []string) {
	dnsclient := new(dns.Client)
	dnsclient.Net = "udp"

	if stateTCP {
		dnsclient.Net = "tcp"
	}

	if stateIPV4 {
		dnsclient.Net += "4"
	}

	if stateIPV6 {
		dnsclient.Net += "6"
	}

	dnsclient.Timeout = stateDNSTimeout

	dnsType := dns.StringToType[string(recordtype)]

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(fqdn), dnsType)

	in, _, err := dnsclient.Exchange(msg, ddrmAppConfig.DnsServer1)

	if err != nil {
		dbgf(ddrmReportDNSClientErr, ddrmAppConfig.DnsServer1, fqdn, recordtype, err)

		// try the second configured DNS server if it's configured
		if len(ddrmAppConfig.DnsServer2) > 0 {
			in, _, err = dnsclient.Exchange(msg, ddrmAppConfig.DnsServer2)
		}

		if err != nil {
			return nil
		}
	}

	answer = []string{}

	for _, r := range in.Answer {
		switch t := r.(type) {
		case *dns.A:
			answer = append(answer, stringProcessor(t.A.String()))
		case *dns.AAAA:
			answer = append(answer, stringProcessor(t.AAAA.String()))
		case *dns.CNAME:
			answer = append(answer, stringProcessor(t.Target))
		case *dns.CAA:
			answer = append(answer, stringProcessor(t.Value))
		case *dns.MX:
			answer = append(answer, stringProcessor(t.Mx))
		case *dns.TXT:
			for _, txt := range t.Txt {
				answer = append(answer, stringProcessor(txt))
			}
		case *dns.NS:
			answer = append(answer, stringProcessor(t.Ns))
		case *dns.PTR:
			answer = append(answer, stringProcessor(t.Ptr))
		case *dns.SRV:
			answer = append(answer, stringProcessor(t.Target))
		case *dns.SOA:
			answer = append(answer, stringProcessor(t.String()))
		}
	}

	return answer
}

func testDnsClient() {
	if stateDNSClientTest {
		dbgp(getRecordData("sommefeldt.com", "MX"))
		dbgp(getRecordData("sommefeldt.com", "A"))
		dbgp(getRecordData("sommefeldt.com", "SOA"))
		dbgp(getRecordData("sommefeldt.com", "TXT"))
		os.Exit(ddrmExitAfterDNSClientTest)
	}
}
