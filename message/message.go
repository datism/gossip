package message

import (
	"strings"
)

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

// GetHeader returns the values of a specific header
func GetHeader(msg *SIPMessage, header string) ([]string) {
	values, exists := msg.Headers[header]
	if !exists {
		return nil
	} else {
		return values
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

func GetValue(header string) (string) {
	end := strings.Index(header, ";")
	if end == -1 {
		return header 
	} else {
		return header[:end]
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
