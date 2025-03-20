package cseq

import (
	"fmt"
	"strconv"
	"strings"
)

type SIPCseq struct {
	Method string
	Seq    int
}

func Parse(cseq string) (SIPCseq, error) {
	var sip_cseq SIPCseq

	values := strings.SplitN(cseq, " ", 2)
	if seq, err := strconv.Atoi(values[0]); err != nil {
		return sip_cseq, err
	} else {
		sip_cseq.Seq = seq
	}

	sip_cseq.Method = values[1]

	return sip_cseq, nil
}

func (cseq SIPCseq) Serialize() string {
	return fmt.Sprintf("%d %s", cseq.Seq, cseq.Method)
}
