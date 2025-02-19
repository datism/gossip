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

func (cseq SIPCseq) DeepCopy() *SIPCseq {
	return &SIPCseq{
		Method: cseq.Method,
		Seq:    cseq.Seq,
	}
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

func Serialize(cseq *SIPCseq) string {
	return fmt.Sprintf("%d %s", cseq.Seq, cseq.Method)
}
