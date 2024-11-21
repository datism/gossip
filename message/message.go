package message

type Startline struct {
	Request  *Request
	Response *Response
}

// RequestType represents a SIP request
type Request struct {
	Method     string
	RequestURI string
}

// ResponseType represents a SIP response
type Response struct {
	StatusCode   int
	ReasonPhrase string
}

// SIPMessage represents a SIP message
type SIPMessage struct {
	Startline 
	Headers map[string][]string
	Body    []byte
}

// // GetHeaderValues returns the values of a specific header
// func (msg *SIPMessage) GetHeaderValues(header string) []string {
// 	return msg.Headers[strings.ToLower(header)]
// }