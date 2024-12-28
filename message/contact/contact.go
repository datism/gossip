package contact

import (
	"gossip/message/uri"
	"strconv"
	"strings"
)

type SIPContact struct {
	DisName   string
	Uri       *uri.SIPUri
	Qvalue    float32
	Expire    int
	Paras     map[string]string
	Supported []string
}

func Parse(contact string) *SIPContact {
	var sip_contact SIPContact

	sip_contact.Paras = make(map[string]string)
	sip_contact.Supported = make([]string, 0)

	if contact == "*" {
		sip_contact.DisName = "*"
		return &sip_contact
	}

	var dsip_name string
	var addr_spec string
	var params string

	ag_begin := strings.Index(contact, "<")
	if ag_begin == -1 {
		dsip_name = ""
		sc_idx := strings.Index(contact, ";")
		if sc_idx != -1 {
			addr_spec = contact[:sc_idx]
			params = contact[sc_idx+1:]
		} else {
			addr_spec = contact
			params = ""
		}
	} else {
		dsip_name = contact[:ag_begin]
		ag_close := strings.Index(contact, ">")
		if ag_close > ag_begin {
			addr_spec = contact[ag_begin+1 : ag_close]
			if sc_idx := strings.Index(contact[ag_close+1:], ";"); sc_idx != -1 {
				params = contact[ag_close+1+sc_idx+1:]
			} else {
				params = ""
			}
		}
	}

	sip_contact.DisName = dsip_name
	sip_contact.Uri = uri.Parse(addr_spec)
	parseParams(params, &sip_contact)

	return &sip_contact
}

func parseParams(params string, contact *SIPContact) {
	if params == "" {
		return
	}

	for _, kvs := range strings.Split(params, ";") {
		kv := strings.SplitN(kvs, "=", 2)
		if len(kv) == 2 {
			if kv[0] == "q" {
				if qvalue, err := strconv.ParseFloat(kv[1], 32); err == nil {
					contact.Qvalue = float32(qvalue)
				}
			} else if kv[0] == "expires" {
				if expires, err := strconv.Atoi(kv[1]); err == nil {
					contact.Expire = expires
				}
			} else {
				contact.Paras[kv[0]] = kv[1]
			}
		} else if len(kv) == 1 {
			contact.Supported = append(contact.Supported, kv[0])
		}
	}
}
