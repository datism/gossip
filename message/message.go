package message

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gossip/message/contact"
	"gossip/message/cseq"
	"gossip/message/fromto"
	"gossip/message/uri"
	"gossip/message/via"
	"gossip/transport"
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
	From       *fromto.SIPFromTo
	To         *fromto.SIPFromTo
	CSeq       *cseq.SIPCseq
	CallID     string
	Contacts   []*contact.SIPContact
	TopmostVia *via.SIPVia
	Headers    map[string][]string
	Body       []byte
	Transport  *transport.Transport
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

	if from := msg.GetHeader("from"); from != nil {
		msg.From = fromto.Parse(from[0])
		msg.DeleteHeader("from")
	}

	if to := msg.GetHeader("to"); to != nil {
		msg.To = fromto.Parse(to[0])
		msg.DeleteHeader("to")
	}

	if callid := msg.GetHeader("call-id"); callid != nil {
		msg.CallID = callid[0]
		msg.DeleteHeader("call-id")
	}

	if cseq_hdr := msg.GetHeader("cseq"); cseq_hdr != nil {
		msg.CSeq = cseq.Parse(cseq_hdr[0])
		msg.DeleteHeader("cseq")
	}

	if contacts := msg.GetHeader("contact"); contacts != nil {
		msg.Contacts = make([]*contact.SIPContact, 0)
		for _, cont := range contacts {
			msg.Contacts = append(msg.Contacts, contact.Parse(cont))
		}
		msg.DeleteHeader("contact")
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

func Serialize(msg *SIPMessage) []byte {
	var builder strings.Builder

	// Serialize the start line
	if msg.Request != nil {
		builder.WriteString(fmt.Sprintf("%s %s SIP/%s", msg.Request.Method, uri.Serialize(msg.Request.RequestURI), "2.0"))
	} else if msg.Response != nil {
		builder.WriteString(fmt.Sprintf("SIP/%s %d %s", "2.0", msg.Response.StatusCode, msg.Response.ReasonPhrase))
	} else {
		return nil
	}

	// Serialize the From header
	if msg.From != nil {
		builder.WriteString(fmt.Sprintf("\r\nFrom: %s\r\n", fromto.Serialize(msg.From)))
	}

	// Serialize the To header
	if msg.To != nil {
		builder.WriteString(fmt.Sprintf("To: %s\r\n", fromto.Serialize(msg.To)))
	}

	// Serialize the CSeq header
	if msg.CSeq != nil {
		builder.WriteString(fmt.Sprintf("CSeq: %s\r\n", cseq.Serialize(msg.CSeq)))
	}

	// Serialize the Call-ID header
	if msg.CallID != "" {
		builder.WriteString(fmt.Sprintf("Call-ID: %s\r\n", msg.CallID))
	}

	// Serialize the Contact headers
	for _, contc := range msg.Contacts {
		builder.WriteString(fmt.Sprintf("Contact: %s\r\n", contact.Serialize(contc)))
	}

	// Serialize the topmost Via header
	if msg.TopmostVia != nil {
		builder.WriteString(fmt.Sprintf("Via: %s\r\n", via.Serialize(msg.TopmostVia)))
	}

	if vias := msg.GetHeader("via"); vias != nil {
		for _, via := range vias {
			builder.WriteString(fmt.Sprintf("%s: %s\r\n", "via", via))
		}
		msg.DeleteHeader("via")
	}

	// Serialize other headers
	for name, values := range msg.Headers {
		for _, value := range values {
			builder.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
		}
	}

	// Add a blank line to separate headers from the body
	builder.WriteString("\r\n")

	// Serialize the body
	if len(msg.Body) > 0 {
		builder.WriteString(string(msg.Body))
	}

	// Convert the builder content to a byte slice and return
	return []byte(builder.String())
}

func (msg SIPMessage) DeepCopy() *SIPMessage {
	// Deep copy the From field
	var newFrom *fromto.SIPFromTo
	if msg.From != nil {
		newFrom = msg.From.DeepCopy()
	}

	// Deep copy the To field
	var newTo *fromto.SIPFromTo
	if msg.To != nil {
		newTo = msg.To.DeepCopy()
	}

	// Deep copy the CSeq field
	var newCSeq *cseq.SIPCseq
	if msg.CSeq != nil {
		newCSeq = msg.CSeq.DeepCopy()
	}

	// Deep copy the Contacts slice
	var newContacts []*contact.SIPContact
	if msg.Contacts != nil {
		newContacts = make([]*contact.SIPContact, len(msg.Contacts))
		for i, contact := range msg.Contacts {
			if contact != nil {
				newContacts[i] = contact.DeepCopy()
			}
		}
	}

	// Deep copy the TopmostVia field
	var newTopmostVia *via.SIPVia
	if msg.TopmostVia != nil {
		newTopmostVia = msg.TopmostVia.DeepCopy()
	}

	// Deep copy the Headers map
	var newHeaders map[string][]string
	if msg.Headers != nil {
		newHeaders = make(map[string][]string)
		for key, values := range msg.Headers {
			newValues := make([]string, len(values))
			copy(newValues, values)
			newHeaders[key] = newValues
		}
	}

	// Deep copy the Body slice
	var newBody []byte
	if msg.Body != nil {
		newBody = make([]byte, len(msg.Body))
		copy(newBody, msg.Body)
	}

	// Deep copy the Transport field
	var newTransport *transport.Transport
	if msg.Transport != nil {
		newTransport = msg.Transport.DeepCopy()
	}

	// Deep copy the Startline
	var newStartline Startline
	if msg.Request != nil {
		var newRequestURI *uri.SIPUri
		if msg.Request.RequestURI != nil {
			newRequestURI = msg.Request.RequestURI.DeepCopy()
		}

		newStartline.Request = &Request{
			Method:     msg.Request.Method,
			RequestURI: newRequestURI,
		}
	}
	if msg.Response != nil {
		newStartline.Response = &Response{
			StatusCode:   msg.Response.StatusCode,
			ReasonPhrase: msg.Response.ReasonPhrase,
		}
	}

	// Return the new deep copied SIPMessage
	return &SIPMessage{
		Startline:  newStartline,
		From:       newFrom,
		To:         newTo,
		CSeq:       newCSeq,
		CallID:     msg.CallID,
		Contacts:   newContacts,
		TopmostVia: newTopmostVia,
		Headers:    newHeaders,
		Body:       newBody,
		Transport:  newTransport,
	}
}

// GetHeader returns the values of a specific header
func (msg SIPMessage) GetHeader(header string) []string {
	values, exists := msg.Headers[header]
	if !exists {
		return nil
	} else {
		return values
	}
}

func (msg *SIPMessage) AddVia(v *via.SIPVia) {
	msg.Headers["via"] = append(msg.Headers["via"], via.Serialize(msg.TopmostVia))
	msg.TopmostVia = v
}

func (msg *SIPMessage) RemoveVia() {
	msg.TopmostVia = via.Parse(msg.Headers["via"][0])
	msg.Headers["via"] = msg.Headers["via"][1:]
}

func (msg *SIPMessage) AddHeader(header string, value string) {
	msg.Headers[header] = append(msg.Headers[header], value)
}

func (msg *SIPMessage) DeleteHeader(header string) {
	delete(msg.Headers, header)
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
		Startline:  Startline{Response: &Response{StatusCode: status_code, ReasonPhrase: reason}},
		From:       request.From,
		To:         request.To,
		CallID:     request.CallID,
		TopmostVia: request.TopmostVia,
		CSeq:       request.CSeq,
		Headers:    res_hdr,
		Transport:  request.Transport,
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
	if sessionid := inv.GetHeader("session-id"); sessionid != nil {
		ack_hdr["session-id"] = sessionid
	}
	if routes := inv.GetHeader("route"); routes != nil {
		ack_hdr["route"] = routes
	}

	return &SIPMessage{
		Startline: Startline{
			Request: &Request{
				Method:     "ACK",
				RequestURI: inv.Request.RequestURI,
			},
		},
		From:       inv.From,
		CallID:     inv.CallID,
		To:         res.To,
		TopmostVia: inv.TopmostVia,
		CSeq: &cseq.SIPCseq{
			Method: "ACK",
			Seq:    inv.CSeq.Seq,
		},
		Headers:   ack_hdr,
		Transport: inv.Transport,
	}
}
