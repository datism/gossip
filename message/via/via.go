package via

import (
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

func (via SIPVia) DeepCopy() *SIPVia {
	// Deep copy the Opts map
	var newOpts map[string]string
	if via.Opts != nil {
		newOpts = make(map[string]string)
		for key, value := range via.Opts {
			newOpts[key] = value
		}
	}

	// Return the deep copied SIPVia
	return &SIPVia{
		Proto:  via.Proto,
		Domain: via.Domain,
		Port:   via.Port,
		Branch: via.Branch,
		Opts:   newOpts,
	}
}

func Parse(via string) *SIPVia {
	var sip_via SIPVia

	sip_via.Opts = make(map[string]string)

	var sp_pa string
	var sent_proto string
	var sent_by string
	var params string

	space_idx := strings.Index(via, " ")
	if space_idx == -1 {
		return nil
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

	return &sip_via
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
				via.Opts[kv[0]] = kv[1]
			}
		}
	}
}

func Serialize(via *SIPVia) string {
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
	for k, v := range via.Opts {
		result.WriteString(";")
		result.WriteString(k)
		result.WriteString("=")
		result.WriteString(v)
	}

	return result.String()
}
