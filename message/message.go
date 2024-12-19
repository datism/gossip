package message

import (
	"strings"
	"gossip/message/uri"
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
	Opts   map[string][]string
}

type SIPCseq struct {
	Method string
	Seq    int
}

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline
	From     *uri.SIPUri
	To       *uri.SIPUri
	CSeq     SIPCseq
	CallID   string
	Contacts []*uri.SIPUri
	TopostVia SIPVia
	Headers  map[string][]string
	Body     []byte
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
	// Find the position of "param" in the header
	start := strings.Index(header, param)
	if start == -1 {
		// Parameter not found, add it to the header
		return header + ";" + param + "=" + value
	}

	// Move the start position to the beginning of the parameter's value
	start += len(param) + 1

	// Find the next ";" to locate the end of the current param value
	end := strings.Index(header[start:], ";")
	if end == -1 {
		// If no ";" is found, the value is till the end of the string
		return header[:start] + value
	} else {
		// Otherwise, replace the value between the start and end positions
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
