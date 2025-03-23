package sipmess

import (
	"bytes"
	"errors"
	"strconv"
)

type SIPUri struct {
	Scheme  []byte
	User    []byte
	Pass    []byte
	Domain  []byte
	Port    int
	Opts    []byte
	Headers []byte
}

func ParseSipUri(uri []byte) (SIPUri, error) {
	sipURI := SIPUri{Port: -1} // Default port value

	// Find the scheme (e.g., "sip:", "sips:")
	colonIndex := bytes.IndexByte(uri, ':')
	if colonIndex == -1 {
		return sipURI, errors.New("invalid sip uri: missing scheme")
	}
	sipURI.Scheme = uri[:colonIndex]
	rest := uri[colonIndex+1:] // Remaining part

	// Detect if headers exist (separated by '?')
	questionIndex := bytes.IndexByte(rest, '?')
	if questionIndex != -1 {
		sipURI.Headers = rest[questionIndex+1:]
		rest = rest[:questionIndex] // Remove headers part
	}

	// Detect if options exist (separated by ';')
	semiIndex := bytes.IndexByte(rest, ';')
	if semiIndex != -1 {
		sipURI.Opts = rest[semiIndex+1:]
		rest = rest[:semiIndex] // Remove options part
	}

	// Check if user info exists (separated by '@')
	atIndex := bytes.IndexByte(rest, '@')
	var userInfo []byte
	if atIndex != -1 {
		userInfo = rest[:atIndex]
		rest = rest[atIndex+1:] // Remove user info
	}

	// Parse user info (if exists)
	if len(userInfo) > 0 {
		colonIndex = bytes.IndexByte(userInfo, ':')
		if colonIndex == -1 {
			sipURI.User = userInfo
		} else {
			sipURI.User = userInfo[:colonIndex]
			sipURI.Pass = userInfo[colonIndex+1:]
		}
	}

	// Parse domain and port
	colonIndex = bytes.IndexByte(rest, ':')
	if colonIndex == -1 {
		sipURI.Domain = rest
	} else {
		sipURI.Domain = rest[:colonIndex]
		port, err := strconv.Atoi(string(rest[colonIndex+1:]))
		if err != nil {
			return sipURI, errors.New("invalid sip uri: invalid port")
		}
		sipURI.Port = port
	}

	return sipURI, nil
}

func (uri SIPUri) Serialize() []byte {
	// Estimate required capacity to minimize reallocations
	size := len(uri.Scheme) + 1 + len(uri.Domain)
	if uri.User != nil {
		size += len(uri.User) + 1 // For '@'
		if uri.Pass != nil {
			size += len(uri.Pass) + 1 // For ':'
		}
	}
	if uri.Port != -1 {
		size += 1 + 5 // 1 for ':' and up to 5 digits for port
	}
	if uri.Opts != nil {
		size += 1 + len(uri.Opts) // 1 for ';'
	}
	if uri.Headers != nil {
		size += 1 + len(uri.Headers) // 1 for '?'
	}

	// Preallocate slice
	buffer := make([]byte, 0, size)

	// Append scheme
	buffer = append(buffer, uri.Scheme...)
	buffer = append(buffer, ':')

	// Append user and password if present
	if uri.User != nil {
		buffer = append(buffer, uri.User...)
		if uri.Pass != nil {
			buffer = append(buffer, ':')
			buffer = append(buffer, uri.Pass...)
		}
		buffer = append(buffer, '@')
	}

	// Append domain
	buffer = append(buffer, uri.Domain...)

	// Append port if present
	if uri.Port != -1 {
		buffer = append(buffer, ':')
		buffer = strconv.AppendInt(buffer, int64(uri.Port), 10)
	}

	// Append options if present
	if uri.Opts != nil {
		buffer = append(buffer, ';')
		buffer = append(buffer, uri.Opts...)
	}

	// Append headers if present
	if uri.Headers != nil {
		buffer = append(buffer, '?')
		buffer = append(buffer, uri.Headers...)
	}

	return buffer
}
