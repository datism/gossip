package sipmess

import (
  "bytes"
  "fmt"
)

type SIPContact struct {
  DisName []byte
  Uri     SIPUri
  Paras   []byte
}

func ParseSipContact(contact []byte) (SIPContact, error) {
  sipContact := SIPContact{}

  // Handle wildcard contact "*"
  if bytes.Equal(contact, []byte("*")) {
      sipContact.DisName = []byte("*")
      return sipContact, nil
  }

  // Check for display name (anything before '<' is display name)
  start := bytes.IndexByte(contact, '<')
  end := bytes.IndexByte(contact, '>')

  if start != -1 && end != -1 && start < end {
      sipContact.DisName = bytes.TrimSpace(contact[:start])
      uri, err := ParseSipUri(contact[start+1 : end])
      if err != nil {
          return sipContact, fmt.Errorf("parsing SIP URI in contact header %q: %w", contact[start+1:end], err)
      }
      sipContact.Uri = uri
      contact = contact[end+1:] // Move to parameters
  } else {
      // No display name, assume full URI
      uri, err := ParseSipUri(contact)
      if err != nil {
          return sipContact, fmt.Errorf("parsing SIP URI in contact header no display name %q: %w", contact, err)
      }
      sipContact.Uri = uri
      return sipContact, nil
  }

  // Find parameters (starts with ';')
  semiIndex := bytes.IndexByte(contact, ';')
  if semiIndex != -1 {
      sipContact.Paras = contact[semiIndex+1:]
  }

  return sipContact, nil
}

func (contact SIPContact) Serialize() []byte {
  uri := contact.Uri.Serialize()

  // Calculate size of buffer
  size := len(uri)
  if contact.DisName != nil {
      size += len(contact.DisName) + 2 // 2 for '<' and '>'
  }
  if contact.Paras != nil {
      size += 1 + len(contact.Paras) // 1 for ';'
  }

  buffer := make([]byte, 0, size)
  // Serialize display name if exists
  if contact.DisName != nil {
      buffer = append(buffer, contact.DisName...)
      buffer = append(buffer, '<')
  }
  // Serialize URI
  buffer = append(buffer, uri...)
  if contact.DisName != nil {
      buffer = append(buffer, '>')
  }
  // Serialize parameters if exists
  if contact.Paras != nil {
      buffer = append(buffer, ';')
      buffer = append(buffer, contact.Paras...)
  }

  return buffer
}