package contact

import (
	"fmt"
	"gossip/message/uri"
	"strconv"
	"strings"
)

type SIPContact struct {
	DisName   string
	Uri       uri.SIPUri
	Qvalue    float32
	Expire    int
	Paras     map[string]string
	Supported []string
}

func Parse(contact string) (SIPContact, error) {
	var sip_contact SIPContact
	sip_contact.Qvalue = -1
	sip_contact.Expire = -1

	if contact == "*" {
		sip_contact.DisName = "*"
		return sip_contact, nil
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

	contact_uri, err := uri.Parse(addr_spec)
	if err != nil {
		return sip_contact, err
	}
	sip_contact.Uri = contact_uri

	parseParams(params, &sip_contact)

	return sip_contact, nil
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
				if contact.Paras == nil {
					contact.Paras = make(map[string]string)
				}

				contact.Paras[kv[0]] = kv[1]
			}
		} else if len(kv) == 1 {
			if contact.Supported == nil {
				contact.Supported = make([]string, 0)
			}
			contact.Supported = append(contact.Supported, kv[0])
		}
	}
}

func (contact SIPContact) Serialize() string {
	var builder strings.Builder

	// Add Display Name if exists
	if contact.DisName != "" {
		builder.WriteString(fmt.Sprintf("%s ", contact.DisName))
	}

	// Add URI if exists
	builder.WriteString(fmt.Sprintf("<%s>", contact.Uri.Serialize()))

	// Add Parameters
	if contact.Qvalue > 0 {
		builder.WriteString(fmt.Sprintf(";q=%.1f", contact.Qvalue))
	}
	if contact.Expire > 0 {
		builder.WriteString(fmt.Sprintf(";expires=%d", contact.Expire))
	}
	if contact.Paras != nil {
		for k, v := range contact.Paras {
			builder.WriteString(fmt.Sprintf(";%s=%s", k, v))
		}
	}
	if contact.Supported != nil {
		for _, v := range contact.Supported {
			builder.WriteString(fmt.Sprintf(";%s", v))
		}
	}

	return builder.String()
}
