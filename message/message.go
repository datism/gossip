package message

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"gossip/message/uri"
	"io"
	"strconv"
	"strings"
)

type Startline struct {
	Request  *Request
	Response *Response
}

// RequestType represents a SIP request
type Request struct {
	Method     string
	RequestURI *uri.SIPUri
}

// ResponseType represents a SIP response
type Response struct {
	StatusCode   int
	ReasonPhrase string
}

type SIPVia struct {
	Proto  string
	Domain string
	Port   int
	Branch string
	Opts   map[string]string
}

type SIPCseq struct {
	Method string
	Seq    int
}

type SIPFromTo struct {
	Uri   *uri.SIPUri
	Tag   string
	Paras map[string]string
}

type SIPContact struct {
	DisName   string
	Uri       *uri.SIPUri
	Qvalue    float32
	Expire    int
	Paras     map[string]string
	Supported []string
}

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline
	From        *SIPFromTo
	To          *SIPFromTo
	CSeq        *SIPCseq
	CallID      string
	Contacts    []*SIPContact
	topmost_via *SIPVia
	Headers     map[string][]string
	Body        []byte
}

func Parse(data []byte) (*SIPMessage, error) {
	reader := bufio.NewReader(bytes.NewReader(data))

	// Read the start line
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	startLine := strings.TrimSpace(line)

	// Determine if it's a request or response
	var msg SIPMessage
	if strings.HasPrefix(startLine, "SIP/") {
		// It's a response
		var version string
		var statusCode int
		var reasonPhrase string
		if _, err := fmt.Sscanf(startLine, "SIP/%s %d %s", &version, &statusCode, &reasonPhrase); err != nil {
			return nil, err
		}
		msg.Startline.Response = &Response{
			StatusCode:   statusCode,
			ReasonPhrase: reasonPhrase,
		}
	} else {
		// It's a request
		var method, requestURI, version string
		if _, err := fmt.Sscanf(startLine, "%s %s SIP/%s", &method, &requestURI, &version); err != nil {
			return nil, err
		}
		msg.Startline.Request = &Request{
			Method:     method,
			RequestURI: uri.Parse(requestURI),
		}
	}

	// Read headers
	msg.Headers = make(map[string][]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("malformed header line")
			// continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		values := strings.Split(parts[1], ",")
		for _, value := range values {
			// if key == "from" {
			// 	msg.From = ParseFromTo(value)
			// } else if key == "to" {
			// 	msg.To = ParseFromTo(value)
			// } else if key == "call-id" {
			// 	msg.CallID = value
			// } else if key == "cseq" {
			// 	msg.CSeq = ParseCseq(value)
			// } else if key == "contact" {
			// 	msg.Contacts = append(msg.Contacts, ParseContact(value))
			// } else {
			msg.Headers[key] = append(msg.Headers[key], strings.TrimSpace(value))
			// }
		}

		if from := GetHeader(&msg, "from"); from != nil {
			msg.From = ParseFromTo(from[0])
			delete(msg.Headers, "from")
		}

		if to := GetHeader(&msg, "to"); to != nil {
			msg.To = ParseFromTo(to[0])
			delete(msg.Headers, "to")
		}

		if callid := GetHeader(&msg, "call-id"); callid != nil {
			msg.CallID = callid[0]
			delete(msg.Headers, "call-id")
		}

		if cseq := GetHeader(&msg, "cseq"); cseq != nil {
			msg.CSeq = ParseCseq(cseq[0])
			delete(msg.Headers, "cseq")
		}

		if contacts := GetHeader(&msg, "contact"); contacts != nil {
			msg.Contacts = make([]*SIPContact, 0)
			for _, contact := range contacts {
				msg.Contacts = append(msg.Contacts, ParseContact(contact))
			}
			delete(msg.Headers, "contact")
		}

		msg.topmost_via = ParseVia(msg.Headers["via"][0])
	}

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	msg.Body = body

	return &msg, nil
}

func ParseFromTo(fromto string) *SIPFromTo {
	var sip_fromto SIPFromTo

	ag_begin := strings.Index(fromto, "<")
	ag_close := strings.Index(fromto, ">")

	if ag_begin != -1 && ag_close != -1 && ag_begin < ag_close {
		sip_fromto.Uri = uri.Parse(fromto[ag_begin:ag_close])
	} else {
		return nil
	}

	paras := fromto[ag_close+1:]
	if strings.HasPrefix(paras, ";") {
		sip_fromto.Paras = make(map[string]string)
		for _, kvs := range strings.Split(paras[1:], ";") {
			kv := strings.SplitN(kvs, "=", 2)
			if len(kv) == 2 {
				if kv[0] == "branch" {
					sip_fromto.Tag = kv[1]
				}
				sip_fromto.Paras[kv[0]] = kv[1]
			}
		}
	}

	return &sip_fromto
}

func ParseCseq(cseq string) *SIPCseq {
	var sip_cseq SIPCseq

	values := strings.SplitN(cseq, " ", 2)
	if seq, err := strconv.Atoi(values[0]); err == nil {
		return nil
	} else {
		sip_cseq.Seq = seq
	}

	sip_cseq.Method = values[1]

	return &sip_cseq
}

func ParseContact(contact string) *SIPContact {
	var sip_contact SIPContact

	if contact == "*" {
		sip_contact.DisName = "*"
		return &sip_contact
	}

	var dsip_name string
	var addr_spec string
	var params string

	ag_begin := strings.Index(contact, "<")
	if ag_begin != -1 {
		dsip_name = ""
		sc_idx := strings.Index(contact, ";")
		if sc_idx != -1 {
			addr_spec = contact[:sc_idx]
			params = contact[sc_idx+1:]
		} else {
			addr_spec = contact
			params = ""
		}
	} else {
		dsip_name = contact[:ag_begin]
		ag_close := strings.Index(contact, ">")
		if ag_close > ag_begin {
			addr_spec = contact[ag_begin+1 : ag_close]
			if sc_idx := strings.Index(contact[ag_close+1:], ";"); sc_idx != -1 {
				params = contact[ag_close+1+sc_idx:]
			} else {
				params = ""
			}
		}
	}

	sip_contact.DisName = dsip_name
	sip_contact.Uri = uri.Parse(addr_spec)
	ParseContactParams(params, &sip_contact)

	return &sip_contact
}

func ParseContactParams(params string, contact *SIPContact) {
	if params == "" {
		return
	}

	contact.Paras = make(map[string]string)
	contact.Supported = make([]string, 0)

	for _, kvs := range strings.Split(params, ";") {
		kv := strings.SplitN(kvs, "=", 2)
		if len(kv) == 2 {
			if kv[0] == "q" {
				qvalue, err := strconv.ParseFloat(kv[1], 32)
				if err != nil {
					contact.Qvalue = float32(qvalue)
				}
			} else if kv[0] == "expires" {
				expires, err := strconv.Atoi(kv[1])
				if err != nil {
					contact.Expire = expires
				}
			} else {
				contact.Paras[kv[0]] = kv[1]
			}
		} else if len(kv) == 1 {
			contact.Supported = append(contact.Supported, kv[0])
		}
	}
}

func ParseVia(via string) *SIPVia {
	var sip_via SIPVia
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

	ParseViaProto(sent_proto, &sip_via)
	ParseViaSentBy(sent_by, &sip_via)
	ParseViaParams(params, &sip_via)

	return &sip_via
}

func ParseViaProto(proto string, via *SIPVia) {
	strs := strings.SplitN(proto, "/", 3)
	via.Proto = strs[2]
}

func ParseViaSentBy(sentby string, via *SIPVia) {
	colon_idx := strings.Index(sentby, ":")
	if colon_idx != -1 {
		via.Domain = sentby
	} else {
		via.Domain = sentby[:colon_idx]
		if port, err := strconv.Atoi(sentby[colon_idx+1:]); err == nil {
			via.Port = port
		}
	}
}

func ParseViaParams(params string, via *SIPVia) {
	if params == "" {
		return
	}

	via.Opts = make(map[string]string)

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

// GetHeader returns the values of a specific header
func GetHeader(msg *SIPMessage, header string) []string {
	values, exists := msg.Headers[header]
	if !exists {
		return nil
	} else {
		return values
	}
}

func GetValue(header string) string {
	end := strings.Index(header, ";")
	if end == -1 {
		return header
	} else {
		return header[:end]
	}
}

// GetParam returns the values of a specific param of a header
func GetParam(header string, param string) string {
	// Find the position of "param"
	start := strings.Index(header, param)
	if start == -1 {
		return ""
	}

	// Move the start position to the beginning of the value
	start += len(param) + 1

	// Find the next ";" after the branch value
	end := strings.Index(header[start:], ";")
	if end == -1 {
		// No ";" found, return the rest of the string
		return header[start:]
	} else {
		// Extract the substring between start and end
		return header[start : start+end]
	}
}

func GetValueBLWS(str string) string {
	index := strings.Index(str, " ")
	if index == -1 {
		return ""
	} else {
		return str[:index]
	}
}

func GetValueALWS(str string) string {
	index := strings.Index(str, " ")
	if index == -1 {
		return ""
	}
	return str[index+1:]
}

// SetParam sets the value of a specific param in a header, or adds the param if it doesn't exist.
func SetParam(header string, param string, value string) string {
	start := strings.Index(header, param)
	if start == -1 {
		return header + ";" + param + "=" + value
	}

	start += len(param) + 1

	end := strings.Index(header[start:], ";")
	if end == -1 {
		return header[:start] + value
	} else {
		return header[:start] + value + header[start+end:]
	}
}

func MakeGenericResponse(status_code int, reason string, request *SIPMessage) *SIPMessage {
	req_hdr := request.Headers
	res_hdr := make(map[string][]string)

	for _, key := range []string{"via", "session-id"} {
		if value, exists := req_hdr[key]; exists {
			res_hdr[key] = value
		}
	}

	res_hdr["content-length"] = []string{"0"}

	return &SIPMessage{
		Startline: Startline{Response: &Response{StatusCode: status_code, ReasonPhrase: reason}},
		From:      request.From,
		To:        request.To,
		CallID:    request.CallID,
		CSeq:      request.CSeq,
		Headers:   res_hdr,
	}
}

func MakeGenericAck(inv *SIPMessage, res *SIPMessage) *SIPMessage {
	/* RFC 3261 17.1.1.3
		The ACK request constructed by the client transaction MUST contain
	 	values for the Call-ID, From, and Request-URI that are equal to the
	  	values of those header fields in the request passed to the transport
	  	by the client transaction (call this the "original request").

	   	The To header field in the ACK MUST equal the To header field in the
	  	response being acknowledged, and therefore will usually differ from
	  	the To header field in the original request by the addition of the
	  	tag parameter.

	   	The ACK MUST contain a single Via header field, and
	 	this MUST be equal to the top Via header field of the original
	  	request.

	   	The CSeq header field in the ACK MUST contain the same
	  	value for the sequence number as was present in the original request,
	  	but the method parameter MUST be equal to "ACK".

	   	If the INVITE request whose response is being acknowledged had Route
	 	header fields, those header fields MUST appear in the ACK.  This is
	  	to ensure that the ACK can be routed properly through any downstream
	  	stateless proxies.
	*/
	var ack = SIPMessage{
		Startline: Startline{
			Request: &Request{
				Method:     "ACK",
				RequestURI: inv.Request.RequestURI,
			},
		},
	}

	if sessionid := GetHeader(inv, "session-id"); sessionid != nil {
		ack.Headers["session-id"] = sessionid
	}

	ack.From = inv.From
	ack.CallID = inv.CallID

	ack.To = res.To

	vias := GetHeader(inv, "via")
	ack.Headers["vias"] = []string{vias[0]}

	ack.CSeq = SIPCseq{
		Method: "ACK",
		Seq:    inv.CSeq.Seq,
	}

	routes := GetHeader(inv, "route")
	ack.Headers["route"] = routes

	return &ack
}
