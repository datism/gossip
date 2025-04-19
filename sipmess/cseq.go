package sipmess

import (
  "bytes"
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
      return sip_cseq, fmt.Errorf("missing sequence number or method in %q", cseq)
  }

  // Parse method
  meth, err := ParseMethod(cseq[spaceIndex+1:])
  if err != nil {
      return sip_cseq, fmt.Errorf("invalid method in %q: %w", cseq[spaceIndex+1:], err)
  }
  sip_cseq.Method = meth

  // Parse sequence number
  seq, err := strconv.Atoi(string(cseq[:spaceIndex]))
  if err != nil {
      return sip_cseq, fmt.Errorf("invalid sequence number in %q: %w", cseq[:spaceIndex], err)
  }
  sip_cseq.Seq = seq

  return sip_cseq, nil
}

func (cseq SIPCseq) Serialize() []byte {
  return []byte(fmt.Sprintf("%d %s", cseq.Seq, SerializeMethod(cseq.Method)))
}