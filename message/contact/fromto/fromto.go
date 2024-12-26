package fromto

import (
	"gossip/message/uri"
	"strings"
)

type SIPFromTo struct {
	Uri   *uri.SIPUri
	Tag   string
	Paras map[string]string
}

func Parse(fromto string) *SIPFromTo {
	var sip_fromto SIPFromTo

	ag_begin := strings.Index(fromto, "<")
	ag_close := strings.Index(fromto, ">")

	if ag_begin != -1 && ag_begin < ag_close {
		sip_fromto.Uri = uri.Parse(fromto[ag_begin+1:ag_close])
	} else {
		return nil
	}

	paras := fromto[ag_close+1:]
	if strings.HasPrefix(paras, ";") {
		sip_fromto.Paras = make(map[string]string)
		for _, kvs := range strings.Split(paras[1:], ";") {
			kv := strings.SplitN(kvs, "=", 2)
			if len(kv) == 2 {
				if kv[0] == "tag" {
					sip_fromto.Tag = kv[1]
				}
				sip_fromto.Paras[kv[0]] = kv[1]
			}
		}
	}

	return &sip_fromto
}
