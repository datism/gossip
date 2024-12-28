package message

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gossip/message/uri"
	"gossip/message/via"
	"gossip/message/contact"
	"gossip/message/fromto"
	"gossip/message/cseq"
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

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline
	From        *fromto.SIPFromTo
	To          *fromto.SIPFromTo
	CSeq        *cseq.SIPCseq
	CallID      string
	Contacts    []*contact.SIPContact
	TopmostVia *via.SIPVia
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
	}

	
	if from := GetHeader(&msg, "from"); from != nil {
		msg.From = fromto.Parse(from[0])
		delete(msg.Headers, "from")
	}

	if to := GetHeader(&msg, "to"); to != nil {
		msg.To = fromto.Parse(to[0])
		delete(msg.Headers, "to")
	}

	if callid := GetHeader(&msg, "call-id"); callid != nil {
		msg.CallID = callid[0]
		delete(msg.Headers, "call-id")
	}

	if cseq_hdr := GetHeader(&msg, "cseq"); cseq_hdr != nil {
		msg.CSeq = cseq.Parse(cseq_hdr[0])
		delete(msg.Headers, "cseq")
	}

	if contacts := GetHeader(&msg, "contact"); contacts != nil {
		msg.Contacts = make([]*contact.SIPContact, 0)
		for _, cont := range contacts {
			msg.Contacts = append(msg.Contacts, contact.Parse(cont))
		}
		delete(msg.Headers, "contact")
	}

	msg.TopmostVia = via.Parse(msg.Headers["via"][0])
	msg.Headers["via"] = msg.Headers["via"][1:]

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	msg.Body = body

	return &msg, nil
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
		TopmostVia: request.TopmostVia,
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
	ack_hdr := make(map[string][]string)
	if sessionid := GetHeader(inv, "session-id"); sessionid != nil {
		ack_hdr["session-id"] = sessionid
	}
	routes := GetHeader(inv, "route")
	ack_hdr["route"] = routes

	return &SIPMessage{
		Startline: Startline{
			Request: &Request{
				Method:     "ACK",
				RequestURI: inv.Request.RequestURI,
			},
		},
		From : inv.From,
		CallID : inv.CallID,
		To : res.To,
		TopmostVia : inv.TopmostVia,
		CSeq : &cseq.SIPCseq{
			Method: "ACK",
			Seq:    inv.CSeq.Seq,
		},
		Headers: ack_hdr,
	}
}
