package sipmess

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type SIPCseq struct {
	Method SIPMethod
	Seq    int
}

func ParseSipCseq(cseq []byte) (SIPCseq, error) {
	sip_cseq := SIPCseq{Seq: -1} // Default value for Seq

	// Find first space to separate sequence number and method
	spaceIndex := bytes.IndexByte(cseq, ' ')
	if spaceIndex == -1 {
		return sip_cseq, errors.New("invalid CSeq header: missing sequence number or method")
	} else {
		meth, err := ParseMethod(cseq[spaceIndex+1:])
		if err != nil {
			return sip_cseq, errors.Join(err, errors.New("invalid CSeq header"))
		}
		sip_cseq.Method = meth
		seq, err := strconv.Atoi(string(cseq[:spaceIndex]))
		if err != nil {
			return sip_cseq, errors.New("invalid CSeq header: invalid sequence number")
		}
		sip_cseq.Seq = seq
	}

	return sip_cseq, nil
}

func (cseq SIPCseq) Serialize() []byte {
	return []byte(fmt.Sprintf("%d %s", cseq.Seq, SerializeMethod(cseq.Method)))
}
