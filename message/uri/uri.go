package uri

import (
	"strconv"
	"strings"
)

type SIPUri struct {
	Scheme  string
	User    string
	Pass    string
	Domain  string
	Port    int
	Opts    map[string]string
	Headers map[string]string
}

func Parse(uri string) *SIPUri {
	var sip_uri SIPUri

	sip_uri.Opts = make(map[string]string)
	sip_uri.Headers = make(map[string]string)

	var ui_hp_up_hd string
	var hp_up_hd string
	var up_hd string

	var user_info string
	var host_port string
	var opts string
	var headers string

	colon_idx := strings.Index(uri, ":")
	if colon_idx == -1 {
		return nil
	} else {
		scheme := uri[:colon_idx]
		if scheme != "sip" && scheme != "sips" && scheme != "tel" {
			return nil
		} else {
			sip_uri.Scheme = scheme
			ui_hp_up_hd = uri[colon_idx+1:]
		}
	}

	at_idx := strings.Index(ui_hp_up_hd, "@")
	if at_idx == -1 {
		user_info = ""
		hp_up_hd = ui_hp_up_hd
	} else {
		user_info = ui_hp_up_hd[:at_idx]
		hp_up_hd = ui_hp_up_hd[at_idx+1:]
	}

	seco_idx := strings.Index(hp_up_hd, ";")
	if seco_idx == -1 {
		host_port = hp_up_hd
		up_hd = ""
	} else {
		host_port = hp_up_hd[:seco_idx]
		up_hd = hp_up_hd[seco_idx+1:]
	}

	ques_idx := strings.Index(up_hd, "?")
	if ques_idx == -1 {
		opts = up_hd
		headers = ""
	} else {
		opts = up_hd[:ques_idx]
		headers = up_hd[ques_idx+1:]
	}

	parse_user_info(user_info, &sip_uri)
	parse_host_port(host_port, &sip_uri)
	parse_opts(opts, &sip_uri)
	parse_headers(headers, &sip_uri)

	return &sip_uri
}

func parse_user_info(user_info string, uri *SIPUri) {
	if user_info == "" {
		return
	}

	colon_idx := strings.Index(user_info, ":")
	if colon_idx == -1 {
		uri.User = user_info
	} else {
		uri.User = user_info[:colon_idx]
		uri.Pass = user_info[colon_idx+1:]
	}
}

func parse_host_port(host_port string, uri *SIPUri) {
	if host_port == "" {
		return
	}

	colon_idx := strings.Index(host_port, ":")
	if colon_idx == -1 {
		uri.Domain = host_port
		uri.Port = -1
	} else {
		uri.Domain = host_port[:colon_idx]
		if port, err := strconv.Atoi(host_port[colon_idx+1:]); err == nil {
			uri.Port = port
		} else {
			uri.Port = -1
		}
	}
}

func parse_opts(opts string, uri *SIPUri) {
	if opts == "" {
		return
	}

	for _, kvs := range strings.Split(opts, ";") {
		kv := strings.SplitN(kvs, "=", 2)
		if len(kv) == 2 {
			uri.Opts[kv[0]] = kv[1]
		}
	}
}

func parse_headers(headers string, uri *SIPUri) {
	if headers == "" {
		return
	}

	for _, kvs := range strings.Split(headers, "&") {
		kv := strings.SplitN(kvs, "=", 2)
		if len(kv) == 2 {
			uri.Headers[kv[0]] = kv[1]
		}
	}
}
