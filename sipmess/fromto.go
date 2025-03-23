package sipmess

import (
	"bytes"
	"errors"
)

type SIPFromTo struct {
	Uri   SIPUri
	Tag   []byte
	Paras []byte
}

func ParseSipFromTo(input []byte) (SIPFromTo, error) {
	var fromTo SIPFromTo

	// Find parameters (starts with ';')
	semiIndex := bytes.IndexByte(input, ';')
	if semiIndex != -1 {
		fromTo.Paras = input[semiIndex+1:]
		input = input[:semiIndex] // Keep only the URI part for parsing
	}

	// Extract and parse the SIP URI (URI is enclosed in "< >")
	start := bytes.IndexByte(input, '<')
	end := bytes.IndexByte(input, '>')

	if start != -1 && end != -1 && start < end {
		// Extract URI inside "< >"
		sipUri, err := ParseSipUri(input[start+1 : end])
		if err != nil {
			return fromTo, errors.Join(err, errors.New("invalid from to header"))
		}
		fromTo.Uri = sipUri
	} else {
		// If no "< >", assume raw URI
		sipUri, err := ParseSipUri(input)
		if err != nil {
			return fromTo, errors.Join(err, errors.New("invalid from to header"))
		}
		fromTo.Uri = sipUri
	}

	// Extract tag if present
	tagIndex := bytes.Index(fromTo.Paras, []byte("tag="))
	if tagIndex != -1 {
		endIndex := bytes.IndexByte(fromTo.Paras[tagIndex+4:], ';')
		if endIndex == -1 {
			fromTo.Tag = fromTo.Paras[tagIndex+4:]
			fromTo.Paras = fromTo.Paras[:tagIndex]
		} else {
			fromTo.Tag = fromTo.Paras[tagIndex+4 : tagIndex+4+endIndex]
			fromTo.Paras = append(fromTo.Paras[:tagIndex], fromTo.Paras[tagIndex+4+endIndex:]...)
		}
	}

	if len(fromTo.Paras) == 0 {
		fromTo.Paras = nil
	}

	return fromTo, nil
}

func (ft SIPFromTo) Serialize() []byte {
	// Estimate capacity to minimize reallocations
	uri := ft.Uri.Serialize()

	size := len(uri) + 2 // "< >"
	if ft.Tag != nil {
		size += 5 + len(ft.Tag) // ";tag="
	}
	if ft.Paras != nil {
		size += 1 + len(ft.Paras) // ";"
	}

	// Preallocate slice
	buffer := make([]byte, 0, size)

	// Append URI, wrap in "< >" if needed
	buffer = append(buffer, '<')
	buffer = append(buffer, uri...)
	buffer = append(buffer, '>')

	// Append tag if present
	if ft.Tag != nil {
		buffer = append(buffer, ";tag="...)
		buffer = append(buffer, ft.Tag...)
	}

	// Append parameters if present
	if ft.Paras != nil {
		buffer = append(buffer, ';')
		buffer = append(buffer, ft.Paras...)
	}

	return buffer
}
