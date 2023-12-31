package storkutil

import (
	"bytes"
	"strings"

	"github.com/pkg/errors"
)

// Represents Fully Qualified Domain Name (FQDN).
// See https://datatracker.ietf.org/doc/html/rfc1035.
type Fqdn struct {
	// A collection of labels forming the FQDN.
	labels []string
	// Indicates if the FQDN is partial or full.
	partial bool
}

// Returns true if the parsed FQDN is partial. Otherwise
// it returns false.
func (fqdn Fqdn) IsPartial() bool {
	return fqdn.partial
}

// Converts FQDN to bytes form as specified in RFC 1035. It is output
// as a collection of labels, each preceded with a label length.
func (fqdn Fqdn) ToBytes() (buf []byte, err error) {
	var buffer bytes.Buffer
	for _, label := range fqdn.labels {
		if err = buffer.WriteByte(byte(len(label))); err != nil {
			err = errors.WithStack(err)
			return
		}
		if _, err = buffer.WriteString(label); err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	if !fqdn.partial {
		if err = buffer.WriteByte(0); err != nil {
			err = errors.WithStack(err)
			return
		}
	}
	buf = buffer.Bytes()
	return
}

// Parses an FQDN string. If the string does not contain a valid FQDN,
// it returns nil and an error.
func ParseFqdn(fqdn string) (*Fqdn, error) {
	// Remove leading and trailing whitespace.
	fqdn = strings.TrimSpace(fqdn)
	if len(fqdn) == 0 {
		return nil, errors.New("failed to parse an empty FQDN")
	}
	// Full FQDN has a terminating dot.
	full := fqdn[len(fqdn)-1] == '.'
	labels := strings.Split(fqdn, ".")
	if full {
		// If this is a full FQDN, remove last label (after trailing dot).
		labels = labels[:len(labels)-1]
		// Expect that full FQDN has at least 3 labels.
		if len(labels) < 3 {
			return nil, errors.Errorf("full FQDN %s must contain at least three labels", fqdn)
		}
	}
	// Validate the labels.
	for i, label := range labels {
		// Last label in the full FQDN must only contain letters and must be
		// at least two characters long.
		if full && i == len(labels)-1 {
			if len(label) < 2 {
				return nil, errors.Errorf("last label of the full FQDN %s must be at least two characters long", fqdn)
			}
			for _, c := range label {
				if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
					return nil, errors.Errorf("last label of the full FQDN %s must only contain letters and must be at least two characters long", fqdn)
				}
			}
		} else {
			// Other labels must not be empty, may contain digits, letters and hyphens
			// but the hyphens must not be at the start nor at the end of the label.
			if len(label) == 0 {
				return nil, errors.Errorf("empty label found in the FQDN %s", fqdn)
			}
			for i, c := range label {
				if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
					(c < '0' || c > '9') &&
					(i == 0 || i == len(label)-1 || c != '-') {
					return nil, errors.Errorf("first and middle labels in the FQDN %s may only contain digits, letters and hyphens but hyphens must not be at the start and the end of the FQDN", fqdn)
				}
			}
		}
	}
	// Everything good. Create the FQDN instance.
	parsed := &Fqdn{
		labels:  labels,
		partial: !full,
	}
	return parsed, nil
}
