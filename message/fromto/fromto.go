package fromto

import (
	"errors"
	"gossip/message/uri"
	"strings"
)

type SIPFromTo struct {
	Uri   uri.SIPUri
	Tag   string
	Paras map[string]string
}

func Parse(fromto string) (SIPFromTo, error) {
	var sip_fromto SIPFromTo

	ag_begin := strings.Index(fromto, "<")
	ag_close := strings.Index(fromto, ">")

	if ag_begin != -1 && ag_begin < ag_close {
		fromto_uri, err := uri.Parse(fromto[ag_begin+1 : ag_close])

		if err != nil {
			return sip_fromto, err
		} else {
			sip_fromto.Uri = fromto_uri
		}

	} else {
		return sip_fromto, errors.New("invalid From-To header")
	}

	paras := fromto[ag_close+1:]
	if strings.HasPrefix(paras, ";") {
		for _, kvs := range strings.Split(paras[1:], ";") {
			kv := strings.SplitN(kvs, "=", 2)
			if len(kv) == 2 {
				if kv[0] == "tag" {
					sip_fromto.Tag = kv[1]
				} else {
					if sip_fromto.Paras == nil {
						sip_fromto.Paras = make(map[string]string)
					}

					sip_fromto.Paras[kv[0]] = kv[1]
				}
			}
		}
	}

	return sip_fromto, nil
}

func (fromTo SIPFromTo) Serialize() string {
	var result strings.Builder

	// Add URI
	result.WriteString("<")
	result.WriteString(fromTo.Uri.Serialize())
	result.WriteString(">")

	// Add tag if present
	if fromTo.Tag != "" {
		result.WriteString(";tag=")
		result.WriteString(fromTo.Tag)
	}

	// Add other parameters
	if fromTo.Paras != nil {
		for k, v := range fromTo.Paras {
			result.WriteString(";")
			result.WriteString(k)
			result.WriteString("=")
			result.WriteString(v)
		}
	}

	return result.String()
}
