package contact

import (
	"fmt"
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

func (contact *SIPContact) DeepCopy() *SIPContact {
	// Deep copy the Uri field
	var newUri *uri.SIPUri
	if contact.Uri != nil {
		newUri = contact.Uri.DeepCopy() // Use SIPUri's DeepCopy method
	}

	// Deep copy the Paras map
	var newParas map[string]string
	if contact.Paras != nil {
		newParas = make(map[string]string)
		for key, value := range contact.Paras {
			newParas[key] = value
		}
	}

	// Deep copy the Supported slice
	var newSupported []string
	if contact.Supported != nil {
		newSupported = make([]string, len(contact.Supported))
		copy(newSupported, contact.Supported)
	}

	// Return the deep copied SIPContact
	return &SIPContact{
		DisName:   contact.DisName,
		Uri:       newUri,
		Qvalue:    contact.Qvalue,
		Expire:    contact.Expire,
		Paras:     newParas,
		Supported: newSupported,
	}
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

func Serialize(contact *SIPContact) string {
	var builder strings.Builder

	// Add Display Name if exists
	if contact.DisName != "" {
		builder.WriteString(fmt.Sprintf("%s ", contact.DisName))
	}

	// Add URI if exists
	if contact.Uri != nil {
		builder.WriteString(fmt.Sprintf("<%s>", uri.Serialize((contact.Uri))))
	}

	// Add Parameters
	if contact.Qvalue > 0 {
		builder.WriteString(fmt.Sprintf(";q=%.1f", contact.Qvalue))
	}
	if contact.Expire > 0 {
		builder.WriteString(fmt.Sprintf(";expires=%d", contact.Expire))
	}
	for k, v := range contact.Paras {
		builder.WriteString(fmt.Sprintf(";%s=%s", k, v))
	}
	for _, v := range contact.Supported {
		builder.WriteString(fmt.Sprintf(";%s", v))
	}

	return builder.String()
}
