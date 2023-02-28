package keaconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// An interface representing an address pool in Stork, extended with a set
// of Kea-specific pool parameters, such as client classes.
type AddressPool interface {
	dhcpmodel.AddressPoolAccessor
	GetKeaParameters() *PoolParameters
}

// A structure holding Kea-specific pool parameters. Note that the same
// set of parameters is supported for the address and delegated prefix
// pools. This structure should be extended with new Kea-specific parameters
// when they are implemented in Kea.
type PoolParameters struct {
	ClientClassParameters
}

// Represents an address pool structure within a Kea configuration.
type Pool struct {
	ClientClass          string             `mapstructure:"client-class" json:"client-class,omitempty"`
	OptionData           []SingleOptionData `mapstructure:"option-data" json:"option-data,omitempty"`
	Pool                 string             `mapstructure:"pool" json:"pool"`
	RequireClientClasses []string           `mapstructure:"require-client-classes" json:"require-client-classes,omitempty"`
}

// A custom unmarshal function for a Kea address pool. It removes whitespaces from
// the pool range definition. For example: 192.0.2.1 - 192.0.2.10 becomes
// 192.0.2.1-192.0.2.10. If the pool is specified using the prefix form, it converts
// it to the range form.
func (p *Pool) UnmarshalJSON(data []byte) error {
	type t Pool
	if err := json.Unmarshal(data, (*t)(p)); err != nil {
		return err
	}
	if strings.ContainsRune(p.Pool, '-') {
		buf := bytes.Buffer{}
		for i := 0; i < len(p.Pool); i++ {
			if !unicode.IsSpace(rune(p.Pool[i])) {
				buf.WriteByte(p.Pool[i])
			}
		}
		p.Pool = buf.String()
		return nil
	}
	lb, ub, err := storkutil.ParseIPRange(p.Pool)
	if err != nil {
		return errors.Errorf("invalid pool prefix %s", p.Pool)
	}
	p.Pool = fmt.Sprintf("%s-%s", lb, ub)
	return nil
}

// Returns the pool boundaries (lower, upper bounds).
func (p Pool) GetBoundaries() (net.IP, net.IP, error) {
	lb, ub, err := storkutil.ParseIPRange(p.Pool)
	return lb, ub, err
}
