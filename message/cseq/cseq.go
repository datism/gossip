package cseq

import (
	"strconv"
	"strings"
)

type SIPCseq struct {
	Method string
	Seq    int
}

func Parse(cseq string) *SIPCseq {
	var sip_cseq SIPCseq

	values := strings.SplitN(cseq, " ", 2)
	if seq, err := strconv.Atoi(values[0]); err != nil {
		return nil
	} else {
		sip_cseq.Seq = seq
	}

	sip_cseq.Method = values[1]

	return &sip_cseq
}
