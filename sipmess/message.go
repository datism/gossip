package sipmess

import (
	"bytes"
	"fmt"
	"strconv"
)

type Startline struct {
	Request  *Request
	Response *Response
}

// RequestType represents a SIP request
type Request struct {
	Method     SIPMethod
	RequestURI SIPUri
}

func (r *Request) Serialize() []byte {
	meth := SerializeMethod(r.Method)
	ruri := r.RequestURI.Serialize()
	size := len(meth) + 1 + len(ruri) + 8
	buf := make([]byte, 0, size)
	buf = append(buf, meth...)
	buf = append(buf, ' ')
	buf = append(buf, ruri...)
	buf = append(buf, ' ', 'S', 'I', 'P', '/', '2', '.', '0')
	return buf
}

// ResponseType represents a SIP response
type Response struct {
	StatusCode   int
	ReasonPhrase []byte
}

func (r *Response) Serialize() []byte {
	size := 8 + 3 + 1 + len(r.ReasonPhrase)
	buf := make([]byte, 0, size)
	buf = append(buf, 'S', 'I', 'P', '/', '2', '.', '0', ' ')
	buf = append(buf, strconv.Itoa(r.StatusCode)...)
	buf = append(buf, ' ')
	buf = append(buf, r.ReasonPhrase...)
	return buf
}

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline
	From       SIPFromTo
	To         SIPFromTo
	CallID     []byte
	CSeq       SIPCseq
	Contacts   []SIPContact
	TopmostVia SIPVia
	Headers    map[SIPHeader][][]byte
	Body       []byte
	Options    ParseOptions
}

type ParseOptions struct {
	ParseFrom       bool
	ParseTo         bool
	ParseCallID     bool
	ParseCseq       bool
	ParseCseqByType bool
	ParseContacts   bool
	ParseTopMostVia bool
}

func ParseSipMessage(msgRaw []byte, option ParseOptions) (*SIPMessage, error) {
	var msg SIPMessage
	msg.Options = option

	// Split the message into headers and body
	headerEnd := bytes.Index(msgRaw, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return nil, fmt.Errorf("missing header-body separator in %q", msgRaw)
	}
	headersPart := msgRaw[:headerEnd]
	bodyPart := msgRaw[headerEnd+4:]

	// Parse the start line
	lineEnd := bytes.Index(headersPart, []byte("\r\n"))
	if lineEnd == -1 {
		return nil, fmt.Errorf("missing start line in %q", headersPart)
	}
	startLine := headersPart[:lineEnd]
	headersPart = headersPart[lineEnd+2:]

	// Determine if it's a request or response
	if bytes.HasPrefix(startLine, []byte("SIP/")) {
		// It's a response
		parts := bytes.Fields(startLine)
		if len(parts) < 3 {
			return nil, fmt.Errorf("parsing SIP response: invalid start line %q", startLine)
		}
		statusCode, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("parsing SIP response: invalid status code in %q: %w", parts[1], err)
		}
		msg.Startline.Response = &Response{
			StatusCode:   statusCode,
			ReasonPhrase: bytes.Join(parts[2:], []byte(" ")),
		}
	} else {
		// It's a request
		parts := bytes.Fields(startLine)
		if len(parts) < 3 {
			return nil, fmt.Errorf("parsing SIP request: invalid start line %q", startLine)
		}
		requestURI, err := ParseSipUri(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parsing SIP request: invalid request URI in %q: %w", parts[1], err)
		}
		meth, err := ParseMethod(parts[0])
		if err != nil {
			return nil, fmt.Errorf("parsing SIP request %q: %w", parts[0], err)
		}

		msg.Startline.Request = &Request{
			Method:     meth,
			RequestURI: requestURI,
		}
	}

	// Parse headers
	msg.Headers = make(map[SIPHeader][][]byte)
	for len(headersPart) > 0 {
		lineEnd := bytes.Index(headersPart, []byte("\r\n"))
		if lineEnd == -1 {
			break
		}
		line := headersPart[:lineEnd]
		headersPart = headersPart[lineEnd+2:]

		colonIndex := bytes.IndexByte(line, ':')
		if colonIndex == -1 {
			return nil, fmt.Errorf("parsing SIP headers: malformed header line %q", line)
		}

		headerName, err := ParseHeaderName(bytes.TrimSpace(line[:colonIndex]))
		if err != nil {
			return nil, fmt.Errorf("parsing SIP headers %q: %w", line[:colonIndex], err)
		}

		headerValueRaw := bytes.TrimSpace(line[colonIndex+1:])
		headerValues := bytes.Split(headerValueRaw, []byte(","))
		for _, headerValue := range headerValues {
			headerValue = bytes.TrimSpace(headerValue)
			if len(headerValue) == 0 {
				continue
			}
			msg.Headers[headerName] = append(msg.Headers[headerName], headerValue)
		}
	}

	// Parse specific headers based on options
	if option.ParseFrom {
		if fromRaw, ok := msg.Headers[From]; ok {
			from, err := ParseSipFromTo(fromRaw[0])
			if err != nil {
				return nil, fmt.Errorf("parsing From header: %w", err)
			}
			msg.From = from
			delete(msg.Headers, From)
		}
	}

	if option.ParseTo {
		if toRaw, ok := msg.Headers[To]; ok {
			to, err := ParseSipFromTo(toRaw[0])
			if err != nil {
				return nil, fmt.Errorf("parsing To header: %w", err)
			}
			msg.To = to
			delete(msg.Headers, To)
		}
	}

	if option.ParseCallID {
		if callIDRaw, ok := msg.Headers[CallID]; ok {
			msg.CallID = callIDRaw[0]
			delete(msg.Headers, CallID)
		}
	}

	if option.ParseCseq || (option.ParseCseqByType && msg.Response != nil) {
		msg.Options.ParseCseq = true
		if cseqRaw, ok := msg.Headers[CSeq]; ok {
			cseq, err := ParseSipCseq(cseqRaw[0])
			if err != nil {
				return nil, fmt.Errorf("parsing CSeq header: %w", err)
			}
			msg.CSeq = cseq
			delete(msg.Headers, CSeq)
		}
	}

	if option.ParseContacts {
		if contactsRaw, ok := msg.Headers[Contact]; ok {
			for _, contactRaw := range contactsRaw {
				contact, err := ParseSipContact(contactRaw)
				if err != nil {
					return nil, fmt.Errorf("parsing Contact header: %w", err)
				}
				msg.Contacts = append(msg.Contacts, contact)
			}
			delete(msg.Headers, Contact)
		}
	}

	if option.ParseTopMostVia {
		if viaRaw, ok := msg.Headers[Via]; ok {
			topmostVia, err := ParseSipVia(viaRaw[0])
			if err != nil {
				return nil, fmt.Errorf("parsing Via header: %w", err)
			}
			msg.TopmostVia = topmostVia
			msg.Headers[Via] = viaRaw[1:]
		}
	}

	// Assign the body
	msg.Body = bodyPart

	return &msg, nil
}

func (msg SIPMessage) Serialize() []byte {
	var stSr []byte
	if msg.Request != nil {
		stSr = msg.Request.Serialize()
	} else {
		stSr = msg.Response.Serialize()
	}

	var hdrsSr []byte
	if msg.Options.ParseFrom {
		hdrsSr = append(hdrsSr, "From: "...)
		hdrsSr = append(hdrsSr, msg.From.Serialize()...)
		hdrsSr = append(hdrsSr, '\r', '\n')
	}

	if msg.Options.ParseTo {
		hdrsSr = append(hdrsSr, "To: "...)
		hdrsSr = append(hdrsSr, msg.To.Serialize()...)
		hdrsSr = append(hdrsSr, '\r', '\n')
	}

	if msg.Options.ParseCallID {
		hdrsSr = append(hdrsSr, "Call-ID: "...)
		hdrsSr = append(hdrsSr, msg.CallID...)
		hdrsSr = append(hdrsSr, '\r', '\n')
	}

	if msg.Options.ParseCseq {
		hdrsSr = append(hdrsSr, "Cseq: "...)
		hdrsSr = append(hdrsSr, msg.CSeq.Serialize()...)
		hdrsSr = append(hdrsSr, '\r', '\n')
	}

	if msg.Options.ParseContacts {
		for _, contact := range msg.Contacts {
			hdrsSr = append(hdrsSr, "Contact: "...)
			hdrsSr = append(hdrsSr, contact.Serialize()...)
			hdrsSr = append(hdrsSr, '\r', '\n')
		}
	}

	var hasSerializedVia bool
	if msg.Options.ParseTopMostVia {
		hdrsSr = append(hdrsSr, "Via: "...)
		hdrsSr = append(hdrsSr, msg.TopmostVia.Serialize()...)
		hdrsSr = append(hdrsSr, '\r', '\n')

		if vias, ok := msg.Headers[Via]; ok {
			for _, via := range vias {
				hdrsSr = append(hdrsSr, "Via: "...)
				hdrsSr = append(hdrsSr, via...)
				hdrsSr = append(hdrsSr, '\r', '\n')
			}
		}

		hasSerializedVia = true
	}

	for hdr, vals := range msg.Headers {
		var hdrSr []byte

		if hdr == Via && hasSerializedVia {
			continue
		}

		hdrNameSr := SerializeHeaderName(hdr)
		for _, val := range vals {
			hdrSr = append(hdrSr, hdrNameSr...)
			hdrSr = append(hdrSr, ':')
			hdrSr = append(hdrSr, val...)
			hdrSr = append(hdrSr, '\r', '\n')
		}
		hdrsSr = append(hdrsSr, hdrSr...)
	}

	var msgSr []byte
	msgSr = append(msgSr, stSr...)
	msgSr = append(msgSr, '\r', '\n')
	msgSr = append(msgSr, hdrsSr...)
	msgSr = append(msgSr, '\r', '\n')
	msgSr = append(msgSr, msg.Body...)
	return msgSr
}

// GetHeader returns the values of a specific header
func (msg SIPMessage) GetHeader(header SIPHeader) [][]byte {
	values, exists := msg.Headers[header]
	if !exists {
		return nil
	} else {
		return values
	}
}

func (msg *SIPMessage) AddVia(v SIPVia) {
	msg.Headers[Via] = append(msg.Headers[Via], msg.TopmostVia.Serialize())
	msg.TopmostVia = v
}

func (msg *SIPMessage) DeleteVia() {
	top_most_via, err := ParseSipVia(msg.Headers[Via][0])
	if err != nil {
		return
	}
	msg.TopmostVia = top_most_via
	msg.Headers[Via] = msg.Headers[Via][1:]
}

func (msg *SIPMessage) AddHeader(header SIPHeader, value []byte) {
	msg.Headers[header] = append(msg.Headers[header], value)
}

func (msg *SIPMessage) DeleteHeader(header SIPHeader) {
	delete(msg.Headers, header)
}

// func GetValue(header string) string {
// 	end := strings.Index(header, ";")
// 	if end == -1 {
// 		return header
// 	} else {
// 		return header[:end]
// 	}
// }

// // GetParam returns the values of a specific param of a header
// func GetParam(header string, param string) string {
// 	// Find the position of "param"
// 	start := strings.Index(header, param)
// 	if start == -1 {
// 		return ""
// 	}

// 	// Move the start position to the beginning of the value
// 	start += len(param) + 1

// 	// Find the next ";" after the branch value
// 	end := strings.Index(header[start:], ";")
// 	if end == -1 {
// 		// No ";" found, return the rest of the string
// 		return header[start:]
// 	} else {
// 		// Extract the substring between start and end
// 		return header[start : start+end]
// 	}
// }

// func GetValueBLWS(str string) string {
// 	index := strings.Index(str, " ")
// 	if index == -1 {
// 		return ""
// 	} else {
// 		return str[:index]
// 	}
// }

// func GetValueALWS(str string) string {
// 	index := strings.Index(str, " ")
// 	if index == -1 {
// 		return ""
// 	}
// 	return str[index+1:]
// }

// // SetParam sets the value of a specific param in a header, or adds the param if it doesn't exist.
// func SetParam(header string, param string, value string) string {
// 	start := strings.Index(header, param)
// 	if start == -1 {
// 		return header + ";" + param + "=" + value
// 	}

// 	start += len(param) + 1

// 	end := strings.Index(header[start:], ";")
// 	if end == -1 {
// 		return header[:start] + value
// 	} else {
// 		return header[:start] + value + header[start+end:]
// 	}
// }
