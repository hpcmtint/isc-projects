package dbmodel

import (
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Interface checks.
var (
	_ dhcpmodel.DHCPOptionAccessor      = (*DHCPOption)(nil)
	_ dhcpmodel.DHCPOptionFieldAccessor = (*DHCPOptionField)(nil)
)

// The common part of the structures that contain the DHCP option set.
type DHCPOptionSet struct {
	Options []DHCPOption `pg:"dhcp_option_set"`
	Hash    string       `pg:"dhcp_option_set_hash"`
}

func NewDHCPOptionSet(options []DHCPOption) DHCPOptionSet {
	instance := DHCPOptionSet{}
	instance.SetDHCPOptions(options)
	return instance
}

func (*DHCPOptionSet) calculateHash(options []DHCPOption) string {
	if len(options) == 0 {
		return ""
	}

	// Calculate hash.
	hash := storkutil.Fnv128(options)

	return hash
}

func (s *DHCPOptionSet) SetDHCPOptions(options []DHCPOption) {
	s.Options = options
	s.Hash = s.calculateHash(options)
}

func (s *DHCPOptionSet) Equals(other *DHCPOptionSet) bool {
	return s.Hash == other.Hash
}

// Represents a DHCP option field.
type DHCPOptionField struct {
	FieldType string
	Values    []interface{}
}

// Represents a DHCP option.
type DHCPOption struct {
	AlwaysSend  bool
	Code        uint16
	Encapsulate string
	Fields      []DHCPOptionField
	Name        string
	Space       string
	Universe    storkutil.IPType
}

// Returns option field type.
func (field DHCPOptionField) GetFieldType() string {
	return field.FieldType
}

// Returns option field values.
func (field DHCPOptionField) GetValues() []interface{} {
	return field.Values
}

// Checks if the option is always returned to a DHCP client, regardless
// if the client has requested it or not.
func (option DHCPOption) IsAlwaysSend() bool {
	return option.AlwaysSend
}

// Returns option code.
func (option DHCPOption) GetCode() uint16 {
	return option.Code
}

// Returns an encapsulated option space name.
func (option DHCPOption) GetEncapsulate() string {
	return option.Encapsulate
}

// Returns option fields belonging to the option.
func (option DHCPOption) GetFields() (returnedFields []dhcpmodel.DHCPOptionFieldAccessor) {
	for _, field := range option.Fields {
		returnedFields = append(returnedFields, field)
	}
	return returnedFields
}

// Returns option name.
func (option DHCPOption) GetName() string {
	return option.Name
}

// Returns option universe (i.e., IPv4 or IPv6).
func (option DHCPOption) GetUniverse() storkutil.IPType {
	return option.Universe
}

// Returns option space name.
func (option DHCPOption) GetSpace() string {
	return option.Space
}

// Creates DHCP option instance in Stork from a DHCP option in Kea.
func NewDHCPOptionFromKea(optionData keaconfig.SingleOptionData, universe storkutil.IPType, lookup keaconfig.DHCPOptionDefinitionLookup) (*DHCPOption, error) {
	optionAccessor, err := keaconfig.CreateDHCPOption(optionData, universe, lookup)
	if err != nil {
		return nil, err
	}
	option := &DHCPOption{
		AlwaysSend:  optionAccessor.IsAlwaysSend(),
		Code:        optionAccessor.GetCode(),
		Encapsulate: optionAccessor.GetEncapsulate(),
		Name:        optionAccessor.GetName(),
		Space:       optionAccessor.GetSpace(),
		Universe:    optionAccessor.GetUniverse(),
	}
	for _, f := range optionAccessor.GetFields() {
		option.Fields = append(option.Fields, DHCPOptionField{
			FieldType: f.GetFieldType(),
			Values:    f.GetValues(),
		})
	}
	return option, nil
}
