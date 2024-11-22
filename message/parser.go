package message

import (
	"bytes"
	"fmt"
	"errors"
	"io"
	"strings"
	"bufio"
)

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
			RequestURI: requestURI,
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

	// Read body
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	msg.Body = body

	return &msg, nil
}