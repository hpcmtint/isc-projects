package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

// Test that the DHCPOption model implements expected interfaces.
func TestDHCPOptionInterface(t *testing.T) {
	// Create an option.
	option := DHCPOption{
		AlwaysSend:  true,
		Code:        1776,
		Encapsulate: "bar",
		Name:        "foo",
		Space:       "dhcp6",
		Universe:    storkutil.IPv6,
		Fields: []DHCPOptionField{
			{
				FieldType: dhcpmodel.StringField,
				Values:    []interface{}{"bar"},
			},
		},
	}
	// Validate returned values.
	require.True(t, option.IsAlwaysSend())
	require.EqualValues(t, 1776, option.GetCode())
	require.Equal(t, "bar", option.GetEncapsulate())
	require.Equal(t, "foo", option.GetName())
	require.Equal(t, "dhcp6", option.GetSpace())
	require.Equal(t, storkutil.IPv6, option.GetUniverse())
	require.Len(t, option.GetFields(), 1)
	require.Equal(t, dhcpmodel.StringField, option.GetFields()[0].GetFieldType())
	require.Len(t, option.GetFields()[0].GetValues(), 1)
	require.Equal(t, "bar", option.GetFields()[0].GetValues()[0])
}

// Test creating a DHCP option in Stork from a DHCP option in Kea.
func TestNewDHCPOptionFromKea(t *testing.T) {
	optionData := keaconfig.SingleOptionData{
		AlwaysSend: true,
		Code:       23,
		CSVFormat:  true,
		Data:       "8",
		Name:       "option-foo",
		Space:      dhcpmodel.DHCPv4OptionSpace,
	}
	lookup := NewDHCPOptionDefinitionLookup()
	option, err := NewDHCPOptionFromKea(optionData, storkutil.IPv4, lookup)
	require.NoError(t, err)
	require.NotNil(t, option)

	require.True(t, option.IsAlwaysSend())
	require.EqualValues(t, 23, option.GetCode())
	require.Equal(t, dhcpmodel.DHCPv4OptionSpace, option.GetSpace())
	require.Equal(t, "option-foo", option.GetName())
	fields := option.GetFields()
	require.Len(t, fields, 1)
	require.Equal(t, "uint8", fields[0].GetFieldType())
	require.Len(t, fields[0].GetValues(), 1)
	require.EqualValues(t, 8, fields[0].GetValues()[0])
}

// Test that the hash value is not affected by the name of the option.
func TestDHCPOptionSetHashIgnoreName(t *testing.T) {
	// Arrange
	optionSet := DHCPOptionSet{}

	// Act
	optionSet.SetDHCPOptions([]DHCPOption{{}})
	noNameHash := optionSet.Hash
	optionSet.SetDHCPOptions([]DHCPOption{{Name: "foo"}})
	withNameHash := optionSet.Hash

	// Assert
	require.Equal(t, noNameHash, withNameHash)
}
