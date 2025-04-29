package sipmess

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type SIPVia struct {
	Tranport string
	Domain   []byte
	Port     int
	Branch   []byte
	Opts     []byte
}

func ParseSipVia(via []byte) (SIPVia, error) {
	sipVia := SIPVia{Port: -1} // Default value for Port

	// Find first space to separate protocol
	spaceIndex := bytes.IndexByte(via, ' ')
	if spaceIndex == -1 {
		return sipVia, fmt.Errorf("missing protocol or domain in %q", via)
	}

	proto := string(via[:spaceIndex])
	// Extract transport from protocol (e.g., "SIP/2.0/UDP")
	protoParts := bytes.Split([]byte(proto), []byte("/"))
	if len(protoParts) < 3 {
		return sipVia, fmt.Errorf("invalid protocol format in %q", proto)
	}
	sipVia.Tranport = strings.ToLower(string(protoParts[2]))

	rest := via[spaceIndex+1:] // Remaining part

	// Find first semicolon to split domain and options
	semiIndex := bytes.IndexByte(rest, ';')
	var domainPart []byte

	if semiIndex == -1 {
		// No options, everything after space is domain
		domainPart = rest
	} else {
		// Split domain and options
		domainPart = rest[:semiIndex]
		sipVia.Opts = rest[semiIndex+1:] // Options start after semicolon
	}

	// Find port (split by colon)
	colonIndex := bytes.IndexByte(domainPart, ':')
	if colonIndex == -1 {
		sipVia.Domain = domainPart
	} else {
		sipVia.Domain = domainPart[:colonIndex]
		port, err := strconv.Atoi(string(domainPart[colonIndex+1:]))
		if err != nil {
			return sipVia, fmt.Errorf("invalid port in %q: %w", domainPart[colonIndex+1:], err)
		}
		sipVia.Port = port
	}

	// Extract branch if exists
	if sipVia.Opts != nil {
		branchIndex := bytes.Index(sipVia.Opts, []byte("branch="))
		if branchIndex != -1 {
			// Find end of branch value (delimited by ';' or end of options)
			endIndex := bytes.IndexByte(sipVia.Opts[branchIndex+7:], ';')
			if endIndex == -1 {
				sipVia.Branch = sipVia.Opts[branchIndex+7:]
				sipVia.Opts = sipVia.Opts[:branchIndex] // Remove "branch=" and its value
			} else {
				sipVia.Branch = sipVia.Opts[branchIndex+7 : branchIndex+7+endIndex]
				sipVia.Opts = append(sipVia.Opts[:branchIndex], sipVia.Opts[branchIndex+7+endIndex:]...)
			}
		}

		if len(sipVia.Opts) == 0 {
			sipVia.Opts = nil
		}
	}

	return sipVia, nil
}

func (via SIPVia) Serialize() []byte {
	// Calculate the size of the buffer
	size := 8 + len(via.Tranport) + 1 + len(via.Domain)
	if via.Port != -1 {
		size += 1 + 5 // 1 for ":" and up to 5 digits for the port
	}
	if via.Branch != nil {
		size += 8 + len(via.Branch) // ";branch=" is 8 bytes
	}
	if via.Opts != nil {
		size += len(via.Opts)
	}

	buffer := make([]byte, 0, size)
	// Serialize protocol
	buffer = append(buffer, "SIP/2.0/"...)
	buffer = append(buffer, []byte(strings.ToUpper(via.Tranport))...)
	buffer = append(buffer, ' ')
	// Serialize domain
	buffer = append(buffer, via.Domain...)
	// Serialize port if exists
	if via.Port != -1 {
		buffer = append(buffer, ':')
		buffer = strconv.AppendInt(buffer, int64(via.Port), 10)
	}
	// Serialize branch if exists
	if via.Branch != nil {
		buffer = append(buffer, ";branch="...)
		buffer = append(buffer, via.Branch...)
	}
	// Serialize options if exists
	if via.Opts != nil {
		buffer = append(buffer, via.Opts...)
	}

	return buffer
}
