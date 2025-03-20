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
)

type Startline struct {
	Request  *Request
	Response *Response
}

// RequestType represents a SIP request
type Request struct {
	Method     string
	RequestURI uri.SIPUri
}

// ResponseType represents a SIP response
type Response struct {
	StatusCode   int
	ReasonPhrase string
}

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline
	From       fromto.SIPFromTo
	To         fromto.SIPFromTo
	CSeq       cseq.SIPCseq
	CallID     string
	Contacts   []contact.SIPContact
	TopmostVia via.SIPVia
	Headers    map[string][]string
	Body       []byte
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

		request_uri, err := uri.Parse(requestURI)
		if err != nil {
			return nil, err
		}

		msg.Startline.Request = &Request{
			Method:     method,
			RequestURI: request_uri,
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
			msg.Headers[key] = append(msg.Headers[key], strings.TrimSpace(value))
		}
	}

	if from_raw := msg.GetHeader("from"); from_raw != nil {
		from, err := fromto.Parse(from_raw[0])
		if err != nil {
			return nil, err
		}
		msg.From = from
		msg.DeleteHeader("from")
	}

	if to_raw := msg.GetHeader("to"); to_raw != nil {
		to, err := fromto.Parse(to_raw[0])
		if err != nil {
			return nil, err
		}
		msg.To = to
		msg.DeleteHeader("to")
	}

	if callid := msg.GetHeader("call-id"); callid != nil {
		msg.CallID = callid[0]
		msg.DeleteHeader("call-id")
	}

	if cseq_raw := msg.GetHeader("cseq"); cseq_raw != nil {
		cseq, err := cseq.Parse(cseq_raw[0])
		if err != nil {
			return nil, err
		}
		msg.CSeq = cseq
		msg.DeleteHeader("cseq")
	}

	if contacts_raw := msg.GetHeader("contact"); contacts_raw != nil {
		msg.Contacts = make([]contact.SIPContact, 0)

		for _, contact_raw := range contacts_raw {
			cont, err := contact.Parse(contact_raw)
			if err == nil {
				msg.Contacts = append(msg.Contacts, cont)
			}
		}
		msg.DeleteHeader("contact")
	}

	top_most_via, err := via.Parse(msg.GetHeader("via")[0])
	if err != nil {
		return nil, err
	}
	msg.TopmostVia = top_most_via
	msg.Headers["via"] = msg.Headers["via"][1:]

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	msg.Body = body

	return &msg, nil
}

func (msg SIPMessage) Serialize() []byte {
	var builder strings.Builder

	// Serialize the start line
	if msg.Request != nil {
		builder.WriteString(fmt.Sprintf("%s %s SIP/%s", msg.Request.Method, msg.Request.RequestURI.Serialize(), "2.0"))
	} else if msg.Response != nil {
		builder.WriteString(fmt.Sprintf("SIP/%s %d %s", "2.0", msg.Response.StatusCode, msg.Response.ReasonPhrase))
	} else {
		return nil
	}

	// Serialize the From header
	builder.WriteString(fmt.Sprintf("\r\nFrom: %s\r\n", msg.From.Serialize()))

	// Serialize the To header
	builder.WriteString(fmt.Sprintf("To: %s\r\n", msg.To.Serialize()))

	// Serialize the CSeq header
	builder.WriteString(fmt.Sprintf("CSeq: %s\r\n", msg.CSeq.Serialize()))

	// Serialize the Call-ID header
	if msg.CallID != "" {
		builder.WriteString(fmt.Sprintf("Call-ID: %s\r\n", msg.CallID))
	}

	// Serialize the Contact headers
	if msg.Contacts != nil {
		for _, contc := range msg.Contacts {
			builder.WriteString(fmt.Sprintf("Contact: %s\r\n", contc.Serialize()))
		}
	}

	// Serialize the topmost Via header
	builder.WriteString(fmt.Sprintf("Via: %s\r\n", msg.TopmostVia.Serialize()))

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

// GetHeader returns the values of a specific header
func (msg SIPMessage) GetHeader(header string) []string {
	values, exists := msg.Headers[header]
	if !exists {
		return nil
	} else {
		return values
	}
}

func (msg *SIPMessage) AddVia(v via.SIPVia) {
	msg.Headers["via"] = append(msg.Headers["via"], msg.TopmostVia.Serialize())
	msg.TopmostVia = v
}

func (msg *SIPMessage) RemoveVia() {
	top_most_via, err := via.Parse(msg.Headers["via"][0])
	if err != nil {
		return
	}
	msg.TopmostVia = top_most_via
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
	}
}
