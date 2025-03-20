package via

import (
	"errors"
	"strconv"
	"strings"
)

type SIPVia struct {
	Proto  string
	Domain string
	Port   int
	Branch string
	Opts   map[string]string
}

func Parse(via string) (SIPVia, error) {
	var sip_via SIPVia

	var sp_pa string
	var sent_proto string
	var sent_by string
	var params string

	space_idx := strings.Index(via, " ")
	if space_idx == -1 {
		return sip_via, errors.New("invalid Via header")
	} else {
		sent_proto = via[:space_idx]
		sp_pa = via[space_idx+1:]
	}

	sc_idx := strings.Index(sp_pa, ";")
	if sc_idx != -1 {
		sent_by = sp_pa[:sc_idx]
		params = sp_pa[sc_idx+1:]
	} else {
		sent_by = sp_pa
		params = ""
	}

	parseProto(sent_proto, &sip_via)
	parseSentBy(sent_by, &sip_via)
	parseParams(params, &sip_via)

	return sip_via, nil
}

func parseProto(proto string, via *SIPVia) {
	strs := strings.SplitN(proto, "/", 3)
	via.Proto = strs[2]
}

func parseSentBy(sentby string, via *SIPVia) {
	colon_idx := strings.Index(sentby, ":")
	if colon_idx == -1 {
		via.Domain = sentby
	} else {
		via.Domain = sentby[:colon_idx]
		if port, err := strconv.Atoi(sentby[colon_idx+1:]); err == nil {
			via.Port = port
		}
	}
}

func parseParams(params string, via *SIPVia) {
	if params == "" {
		return
	}

	for _, kvs := range strings.Split(params, ";") {
		kv := strings.SplitN(kvs, "=", 2)
		if len(kv) == 2 {
			if kv[0] == "branch" {
				via.Branch = kv[1]
			} else {
				if via.Opts == nil {
					via.Opts = make(map[string]string)
				}

				via.Opts[kv[0]] = kv[1]
			}
		}
	}
}

func (via SIPVia) Serialize() string {
	var result strings.Builder

	// Add protocol
	if via.Proto != "" {
		result.WriteString("SIP/2.0/")
		result.WriteString(via.Proto)
	}

	// Add domain and port
	result.WriteString(" ")
	result.WriteString(via.Domain)
	if via.Port > 0 {
		result.WriteString(":")
		result.WriteString(strconv.Itoa(via.Port))
	}

	// Add branch parameter
	if via.Branch != "" {
		result.WriteString(";branch=")
		result.WriteString(via.Branch)
	}

	// Add other options
	if via.Opts != nil {
		for k, v := range via.Opts {
			result.WriteString(";")
			result.WriteString(k)
			result.WriteString("=")
			result.WriteString(v)
		}
	}

	return result.String()
}
