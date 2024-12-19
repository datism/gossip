package uri

import (
	"strings"
)

type SIPUri struct {
	Scheme  string
	User    string
	Pass    string
	Domain  string
	Port    int
	Opts    map[string][]string
	ExtOpts map[string][]string
}

func Parse(raw_uri string) *SIPUri {
	var sip_uri SIPUri
	sip_uri.Opts = make(map[string][]string)

	var ui_hp_up string
	var ui_hp string
	var user_info string
	var host_port string
	var opts string
	var ex_opts string

	if strings.HasPrefix(raw_uri, "<") {
		closing_angle_idx := strings.LastIndex(raw_uri, ">")
		if closing_angle_idx == -1 {
			return nil
		}

		ui_hp_up = raw_uri[1:closing_angle_idx]
		ex_opts = raw_uri[closing_angle_idx+1:]
	} else {
		ui_hp_up = ""
		sc_idx := strings.Index(raw_uri, ";")
		if sc_idx == -1 {
			ui_hp = raw_uri
			ex_opts = ""
		} else {
			ui_hp = raw_uri[1:sc_idx]
			ex_opts = raw_uri[sc_idx:]
		}
	}

	if ui_hp_up != "" {
		sc_idx := strings.Index(raw_uri, ";")
		if sc_idx == -1 {
			ui_hp = ui_hp_up
			opts = ""
		} else {
			ui_hp = ui_hp_up[1:sc_idx]
			opts = ui_hp_up[sc_idx:]
		}
	}

	at_idx := strings.Index(ui_hp, "@")
	if at_idx == -1 {
		host_port = ui_hp
		user_info = ""
	} else {
		user_info = ui_hp[1:at_idx]
		host_port = ui_hp[at_idx:]
	}

	parse_user_info(user_info, &sip_uri)
	parse_host_port(host_port, &sip_uri)
	parse_opts(opts, &sip_uri)
	parse_ext_opts(ex_opts, &sip_uri)

	return &sip_uri
}

func parse_user_info(user_info string, uri *SIPUri) {

}

func parse_host_port(host_port string, uri *SIPUri) {

}

func parse_opts(opts string, uri *SIPUri) {

}

func parse_ext_opts(ex_opts string, uri *SIPUri) {

}
