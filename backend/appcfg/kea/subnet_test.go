package keaconfig_test

import (
	"encoding/json"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	keaconfig "isc.org/stork/appcfg/kea"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=keaconfig_test -destination=subnetmock_test.go isc.org/stork/appcfg/kea Subnet

// Alias for the storkutil.Ptr function.
func ptr[T any](value T) *T {
	return storkutil.Ptr(value)
}

// Returns a JSON structure with all configurable IPv4 subnet parameters in Kea.
// It has been initially created from the Kea's all-keys.json file and then
// slightly modified.
func getAllKeysSubnet4() string {
	return `
	{
		"4o6-interface": "eth1",
		"4o6-interface-id": "ethx",
		"4o6-subnet": "2001:db8:1:1::/64",
		"allocator": "iterative",
		"authoritative": false,
		"boot-file-name": "/tmp/boot",
		"client-class": "foobar",
		"ddns-generated-prefix": "myhost",
		"ddns-override-client-update": true,
		"ddns-override-no-update": true,
		"ddns-qualifying-suffix": "example.org",
		"ddns-replace-client-name": "never",
		"ddns-send-updates": true,
		"ddns-update-on-renew": true,
		"ddns-use-conflict-resolution": true,
		"hostname-char-replacement": "x",
		"hostname-char-set": "[^A-Za-z0-9.-]",
		"id": 1,
		"interface": "eth0",
		"match-client-id": true,
		"next-server": "0.0.0.0",
		"store-extended-info": true,
		"option-data": [
			{
				"always-send": true,
				"code": 3,
				"csv-format": true,
				"data": "192.0.3.1",
				"name": "routers",
				"space": "dhcp4"
			}
		],
		"pools": [
			{
				"client-class": "phones_server1",
				"option-data": [],
				"pool": "192.1.0.1 - \t192.1.0.200",
				"require-client-classes": [ "late" ]
			},
			{
				"client-class": "phones_server2",
				"option-data": [],
				"pool": "192.3.0.1 - 192.3.0.200",
				"require-client-classes": []
			}
		],
		"rebind-timer": 40,
		"relay": {
			"ip-addresses": [
				"192.168.56.1"
			]
		},
		"renew-timer": 30,
		"reservations-global": true,
		"reservations-in-subnet": true,
		"reservations-out-of-pool": true,
		"calculate-tee-times": true,
		"t1-percent": 0.5,
		"t2-percent": 0.75,
		"cache-threshold": 0.25,
		"cache-max-age": 1000,
		"reservations": [
			{
				"circuit-id": "01:11:22:33:44:55:66",
				"ip-address": "192.0.2.204",
				"hostname": "foo.example.org",
				"option-data": [
					{
						"name": "vivso-suboptions",
						"data": "4491"
					}
				]
			}
		],
		"require-client-classes": [ "late" ],
		"server-hostname": "myhost.example.org",
		"subnet": "192.0.0.0/8",
		"valid-lifetime": 6000,
		"min-valid-lifetime": 4000,
		"max-valid-lifetime": 8000
	}
`
}

// Returns a JSON structure with all configurable IPv6 subnet parameters in Kea.
// It has been initially created from the Kea's all-keys.json file and then
// slightly modified.
func getAllKeysSubnet6() string {
	return `
	{
		"allocator": "iterative",
		"pd-allocator": "iterative",
		"client-class": "foobar",
		"ddns-generated-prefix": "myhost",
		"ddns-override-client-update": true,
		"ddns-override-no-update": true,
		"ddns-qualifying-suffix": "example.org",
		"ddns-replace-client-name": "never",
		"ddns-send-updates": true,
		"ddns-update-on-renew": true,
		"ddns-use-conflict-resolution": true,
		"hostname-char-replacement": "x",
		"hostname-char-set": "[^A-Za-z0-9.-]",
		"id": 1,
		"interface": "eth0",
		"interface-id": "ethx",
		"store-extended-info": true,
		"option-data": [
			{
				"always-send": true,
				"code": 7,
				"csv-format": true,
				"data": "15",
				"name": "preference",
				"space": "dhcp6"
			}
		],
		"pd-pools": [
			{
				"client-class": "phones_server1",
				"delegated-len": 64,
				"excluded-prefix": "2001:db8:1::",
				"excluded-prefix-len": 72,
				"option-data": [],
				"prefix": "2001:db8:1::",
				"prefix-len": 48,
				"require-client-classes": []
			}
		],
		"pools": [
			{
				"client-class": "phones_server1",
				"option-data": [],
				"pool": "2001:db8:0:1::/64",
				"require-client-classes": [ "late" ]
			},
			{
				"client-class": "phones_server2",
				"option-data": [],
				"pool": "2001:db8:0:3::/64",
				"require-client-classes": []
			}
		],
		"preferred-lifetime": 2000,
		"min-preferred-lifetime": 1500,
		"max-preferred-lifetime": 2500,
		"rapid-commit": true,
		"rebind-timer": 40,
		"relay": {
			"ip-addresses": [
				"2001:db8:0:f::1"
			]
		},
		"renew-timer": 30,
		"reservations-global": true,
		"reservations-in-subnet": true,
		"reservations-out-of-pool": true,
		"calculate-tee-times": true,
		"t1-percent": 0.5,
		"t2-percent": 0.75,
		"cache-threshold": 0.25,
		"cache-max-age": 10,
		"reservations": [
			{
				"duid": "01:02:03:04:05:06:07:08:09:0A",
				"ip-addresses": [ "2001:db8:1:cafe::1" ],
				"prefixes": [ "2001:db8:2:abcd::/64" ],
				"hostname": "foo.example.com",
				"option-data": [
					{
						"name": "vendor-opts",
						"data": "4491"
					}
				]
			}
		],
		"require-client-classes": [ "late" ],
		"subnet": "2001:db8::/32",
		"valid-lifetime": 6000,
		"min-valid-lifetime": 4000,
		"max-valid-lifetime": 8000
	}
`
}

// Test that Kea subnet configuration is properly decoded into the
// keaconfig.Subnet4 structure.
func TestDecodeAllKeysSubnet4(t *testing.T) {
	params := keaconfig.Subnet4{}
	err := json.Unmarshal([]byte(getAllKeysSubnet4()), &params)
	require.NoError(t, err)

	require.Equal(t, "eth1", *params.FourOverSixInterface)
	require.Equal(t, "ethx", *params.FourOverSixInterfaceID)
	require.Equal(t, "2001:db8:1:1::/64", *params.FourOverSixSubnet)
	require.Equal(t, "iterative", *params.Allocator)
	require.Equal(t, "/tmp/boot", *params.BootFileName)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnReview)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.EqualValues(t, 1, params.ID)
	require.Equal(t, "eth0", *params.Interface)
	require.True(t, *params.MatchClientID)
	require.Equal(t, "0.0.0.0", *params.NextServer)
	require.True(t, *params.StoreExtendedInfo)
	require.Len(t, params.OptionData, 1)
	require.True(t, params.OptionData[0].AlwaysSend)
	require.EqualValues(t, 3, params.OptionData[0].Code)
	require.True(t, params.OptionData[0].CSVFormat)
	require.Equal(t, "192.0.3.1", params.OptionData[0].Data)
	require.Equal(t, "routers", params.OptionData[0].Name)
	require.Equal(t, "dhcp4", params.OptionData[0].Space)
	require.Len(t, params.Pools, 2)
	require.Equal(t, "phones_server1", params.Pools[0].ClientClass)
	require.Empty(t, params.Pools[0].OptionData)
	require.Equal(t, "192.1.0.1-192.1.0.200", params.Pools[0].Pool)
	require.Len(t, params.Pools[0].RequireClientClasses, 1)
	require.Equal(t, "phones_server2", params.Pools[1].ClientClass)
	require.Empty(t, params.Pools[1].OptionData)
	require.Equal(t, "192.3.0.1-192.3.0.200", params.Pools[1].Pool)
	require.Empty(t, params.Pools[1].RequireClientClasses)
	require.EqualValues(t, 40, *params.RebindTimer)
	require.Len(t, params.Relay.IPAddresses, 1)
	require.Equal(t, "192.168.56.1", params.Relay.IPAddresses[0])
	require.EqualValues(t, 30, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 1000, *params.CacheMaxAge)
	require.Len(t, params.Reservations, 1)
	require.Equal(t, "01:11:22:33:44:55:66", params.Reservations[0].CircuitID)
	require.Equal(t, "192.0.2.204", params.Reservations[0].IPAddress)
	require.Equal(t, "foo.example.org", params.Reservations[0].Hostname)
	require.Len(t, params.Reservations[0].OptionData, 1)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.Equal(t, "myhost.example.org", *params.ServerHostname)
	require.Equal(t, "192.0.0.0/8", params.Subnet)
	require.EqualValues(t, 6000, *params.ValidLifetime)
	require.EqualValues(t, 4000, *params.MinValidLifetime)
	require.EqualValues(t, 8000, *params.MaxValidLifetime)
}

// Test getting a canonical subnet prefix when the prefix is already in
// that form.
func TestGetCanonicalPrefixMandatorySubnetParameters(t *testing.T) {
	params := keaconfig.MandatorySubnetParameters{
		Subnet: "192.0.2.0/24",
	}
	prefix, err := params.GetCanonicalPrefix()
	require.NoError(t, err)
	require.Equal(t, "192.0.2.0/24", prefix)
}

// Test that an error is reported during getting the canonical prefix when
// the prefix is invalid.
func TestGetCanonicalPrefixMandatorySubnetParametersInvalidPrefix(t *testing.T) {
	params := keaconfig.MandatorySubnetParameters{
		Subnet: "foo",
	}
	_, err := params.GetCanonicalPrefix()
	require.Error(t, err)
}

// Test getting a canonical prefix for an IPv4 subnet.
func TestGetCanonicalPrefixSubnet4(t *testing.T) {
	subnet4 := keaconfig.Subnet4{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "192.0.2.1/24",
		},
	}
	prefix, err := subnet4.GetCanonicalPrefix()
	require.NoError(t, err)
	require.Equal(t, "192.0.2.0/24", prefix)
}

// Test getting a canonical prefix for an IPv6 subnet.
func TestGetCanonicalPrefixSubnet6(t *testing.T) {
	subnet6 := keaconfig.Subnet6{
		MandatorySubnetParameters: keaconfig.MandatorySubnetParameters{
			Subnet: "2001:db8:1:0:0::/64",
		},
	}
	prefix, err := subnet6.GetCanonicalPrefix()
	require.NoError(t, err)
	require.Equal(t, "2001:db8:1::/64", prefix)
}

// Test that the Kea IPv4 subnet configuration parameters are returned
// in the keaconfig.SubnetParameters union.
func TestGetParametersSubnet4(t *testing.T) {
	subnet4 := keaconfig.Subnet4{}
	err := json.Unmarshal([]byte(getAllKeysSubnet4()), &subnet4)
	require.NoError(t, err)

	params := *subnet4.GetSubnetParameters()
	require.NotNil(t, params)

	require.Equal(t, "eth1", *params.FourOverSixInterface)
	require.Equal(t, "ethx", *params.FourOverSixInterfaceID)
	require.Equal(t, "2001:db8:1:1::/64", *params.FourOverSixSubnet)
	require.Equal(t, "iterative", *params.Allocator)
	require.Equal(t, "/tmp/boot", *params.BootFileName)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnReview)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.True(t, *params.MatchClientID)
	require.Equal(t, "0.0.0.0", *params.NextServer)
	require.True(t, *params.StoreExtendedInfo)
	require.EqualValues(t, 40, *params.RebindTimer)
	require.Len(t, params.Relay.IPAddresses, 1)
	require.Equal(t, "192.168.56.1", params.Relay.IPAddresses[0])
	require.EqualValues(t, 30, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 1000, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.Equal(t, "myhost.example.org", *params.ServerHostname)
	require.EqualValues(t, 6000, *params.ValidLifetime)
	require.EqualValues(t, 4000, *params.MinValidLifetime)
	require.EqualValues(t, 8000, *params.MaxValidLifetime)
}

// Test that Kea subnet configuration is properly decoded into the
// keaconfig.Subnet6 structure.
func TestDecodeAllKeysSubnet6(t *testing.T) {
	params := keaconfig.Subnet6{}
	err := json.Unmarshal([]byte(getAllKeysSubnet6()), &params)
	require.NoError(t, err)

	require.Equal(t, "iterative", *params.Allocator)
	require.Equal(t, "iterative", *params.PDAllocator)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnReview)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.EqualValues(t, 1, params.ID)
	require.Equal(t, "eth0", *params.Interface)
	require.Equal(t, "ethx", *params.InterfaceID)
	require.True(t, *params.StoreExtendedInfo)
	require.Len(t, params.OptionData, 1)
	require.True(t, params.OptionData[0].AlwaysSend)
	require.EqualValues(t, 7, params.OptionData[0].Code)
	require.True(t, params.OptionData[0].CSVFormat)
	require.Equal(t, "15", params.OptionData[0].Data)
	require.Equal(t, "preference", params.OptionData[0].Name)
	require.Equal(t, "dhcp6", params.OptionData[0].Space)
	require.Len(t, params.Pools, 2)
	require.Equal(t, "phones_server1", params.Pools[0].ClientClass)
	require.Empty(t, params.Pools[0].OptionData)
	require.Equal(t, "2001:db8:0:1::-2001:db8:0:1:ffff:ffff:ffff:ffff", params.Pools[0].Pool)
	require.Len(t, params.Pools[0].RequireClientClasses, 1)
	require.Len(t, params.PDPools, 1)
	require.Equal(t, "phones_server1", params.PDPools[0].ClientClass)
	require.EqualValues(t, 64, params.PDPools[0].DelegatedLen)
	require.Equal(t, "2001:db8:1::", params.PDPools[0].ExcludedPrefix)
	require.EqualValues(t, 72, params.PDPools[0].ExcludedPrefixLen)
	require.Empty(t, params.PDPools[0].OptionData)
	require.Equal(t, "2001:db8:1::", params.PDPools[0].Prefix)
	require.Equal(t, 48, params.PDPools[0].PrefixLen)
	require.Empty(t, params.PDPools[0].RequireClientClasses)
	require.Equal(t, "phones_server2", params.Pools[1].ClientClass)
	require.Empty(t, params.Pools[1].OptionData)
	require.Equal(t, "2001:db8:0:3::-2001:db8:0:3:ffff:ffff:ffff:ffff", params.Pools[1].Pool)
	require.Empty(t, params.Pools[1].RequireClientClasses)
	require.EqualValues(t, 2000, *params.PreferredLifetime)
	require.EqualValues(t, 1500, *params.MinPreferredLifetime)
	require.EqualValues(t, 2500, *params.MaxPreferredLifetime)
	require.True(t, *params.RapidCommit)
	require.EqualValues(t, 40, *params.RebindTimer)
	require.EqualValues(t, 30, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 10, *params.CacheMaxAge)
	require.Len(t, params.Reservations, 1)
	require.Equal(t, "01:02:03:04:05:06:07:08:09:0A", params.Reservations[0].DUID)
	require.Len(t, params.Reservations[0].IPAddresses, 1)
	require.Equal(t, "2001:db8:1:cafe::1", params.Reservations[0].IPAddresses[0])
	require.Equal(t, "foo.example.com", params.Reservations[0].Hostname)
	require.Len(t, params.Reservations[0].OptionData, 1)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.Equal(t, "2001:db8::/32", params.Subnet)
	require.EqualValues(t, 6000, *params.ValidLifetime)
	require.EqualValues(t, 4000, *params.MinValidLifetime)
	require.EqualValues(t, 8000, *params.MaxValidLifetime)
}

// Test that the Kea IPv6 subnet configuration parameters are returned
// in the keaconfig.SubnetParameters union.
func TestGetParametersSubnet6(t *testing.T) {
	subnet6 := keaconfig.Subnet6{}
	err := json.Unmarshal([]byte(getAllKeysSubnet6()), &subnet6)
	require.NoError(t, err)

	params := *subnet6.GetSubnetParameters()
	require.NotNil(t, params)

	require.Equal(t, "iterative", *params.Allocator)
	require.Equal(t, "iterative", *params.PDAllocator)
	require.Equal(t, "foobar", *params.ClientClass)
	require.Equal(t, "myhost", *params.DDNSGeneratedPrefix)
	require.True(t, *params.DDNSOverrideClientUpdate)
	require.True(t, *params.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *params.DDNSQualifyingSuffix)
	require.Equal(t, "never", *params.DDNSReplaceClientName)
	require.True(t, *params.DDNSSendUpdates)
	require.True(t, *params.DDNSUpdateOnReview)
	require.True(t, *params.DDNSUseConflictResolution)
	require.Equal(t, "x", *params.HostnameCharReplacement)
	require.Equal(t, "[^A-Za-z0-9.-]", *params.HostnameCharSet)
	require.Equal(t, "eth0", *params.Interface)
	require.Equal(t, "ethx", *params.InterfaceID)
	require.True(t, *params.StoreExtendedInfo)
	require.EqualValues(t, 2000, *params.PreferredLifetime)
	require.EqualValues(t, 1500, *params.MinPreferredLifetime)
	require.EqualValues(t, 2500, *params.MaxPreferredLifetime)
	require.True(t, *params.RapidCommit)
	require.EqualValues(t, 40, *params.RebindTimer)
	require.EqualValues(t, 30, *params.RenewTimer)
	require.True(t, *params.ReservationsInSubnet)
	require.True(t, *params.ReservationsOutOfPool)
	require.True(t, *params.CalculateTeeTimes)
	require.EqualValues(t, 0.5, *params.T1Percent)
	require.EqualValues(t, 0.75, *params.T2Percent)
	require.EqualValues(t, 0.25, *params.CacheThreshold)
	require.EqualValues(t, 10, *params.CacheMaxAge)
	require.Len(t, params.RequireClientClasses, 1)
	require.Equal(t, "late", params.RequireClientClasses[0])
	require.EqualValues(t, 6000, *params.ValidLifetime)
	require.EqualValues(t, 4000, *params.MinValidLifetime)
	require.EqualValues(t, 8000, *params.MaxValidLifetime)
}

// Test converting an IPv4 subnet in Stork into the subnet configuration
// in Kea.
func TestCreateSubnet4(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a subnet in Stork.
	mock := NewMockSubnet(controller)
	poolMock := NewMockAddressPool(controller)

	// A mock to define an address pool.
	poolMock.EXPECT().GetLowerBound().AnyTimes().Return("192.0.2.10")
	poolMock.EXPECT().GetUpperBound().AnyTimes().Return("192.0.2.20")
	// Return Kea specific pool parameters.
	poolMock.EXPECT().GetKeaParameters().AnyTimes().Return(&keaconfig.PoolParameters{
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("baz"),
			RequireClientClasses: []string{"foo"},
		},
	})
	// Return DHCP options in the pool.
	poolMock.EXPECT().GetDHCPOptions().AnyTimes().Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        6,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp4",
		},
	})
	// Subnet ID.
	mock.EXPECT().GetID(gomock.Any()).Return(int64(5))
	// Subnet prefix.
	mock.EXPECT().GetPrefix().Return("192.0.2.0/24")
	// Return a pool defined above.
	mock.EXPECT().GetAddressPools().Return([]dhcpmodel.AddressPoolAccessor{poolMock})
	// Return subnet-level Kea parameters.
	mock.EXPECT().GetKeaParameters(gomock.Eq(int64(1))).Return(&keaconfig.SubnetParameters{
		CacheParameters: keaconfig.CacheParameters{
			CacheMaxAge:    ptr[int64](1001),
			CacheThreshold: ptr[float32](0.25),
		},
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("myclass"),
			RequireClientClasses: []string{"foo"},
		},
		DDNSParameters: keaconfig.DDNSParameters{
			DDNSGeneratedPrefix:       ptr("example.com"),
			DDNSOverrideClientUpdate:  ptr(true),
			DDNSOverrideNoUpdate:      ptr(true),
			DDNSQualifyingSuffix:      ptr("example.org"),
			DDNSReplaceClientName:     ptr("never"),
			DDNSSendUpdates:           ptr(true),
			DDNSUseConflictResolution: ptr(true),
		},
		FourOverSixParameters: keaconfig.FourOverSixParameters{
			FourOverSixInterface:   ptr("bar-id"),
			FourOverSixInterfaceID: ptr("foo-id"),
			FourOverSixSubnet:      ptr("10.0.0.0/24"),
		},
		HostnameCharParameters: keaconfig.HostnameCharParameters{
			HostnameCharReplacement: ptr("xyz"),
			HostnameCharSet:         ptr("[A-z]"),
		},
		ReservationParameters: keaconfig.ReservationParameters{
			ReservationMode:       ptr("out-of-pool"),
			ReservationsGlobal:    ptr(true),
			ReservationsInSubnet:  ptr(true),
			ReservationsOutOfPool: ptr(true),
		},
		TimerParameters: keaconfig.TimerParameters{
			CalculateTeeTimes: ptr(true),
			RebindTimer:       ptr[int64](300),
			RenewTimer:        ptr[int64](200),
			T1Percent:         ptr[float32](0.32),
			T2Percent:         ptr[float32](0.44),
		},
		ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
			MaxValidLifetime: ptr[int64](1000),
			MinValidLifetime: ptr[int64](500),
			ValidLifetime:    ptr[int64](1001),
		},
		Allocator:     ptr("iterative"),
		Authoritative: ptr(true),
		BootFileName:  ptr("/tmp/bootfile"),
		Interface:     ptr("etx0"),
		InterfaceID:   ptr("id-foo"),
		MatchClientID: ptr(true),
		NextServer:    ptr("192.0.2.1"),
		Relay: &keaconfig.Relay{
			IPAddresses: []string{"10.0.0.1"},
		},
		ServerHostname:    ptr("hostname.example.org"),
		StoreExtendedInfo: ptr(true),
	})
	// Return subnet-level DHCP options.
	mock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        5,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp4",
		},
	})
	// Do not return option definitions. This is not the area of the code
	// that we want to test here.
	lookupMock := NewMockDHCPOptionDefinitionLookup(controller)
	lookupMock.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)

	// Convert the subnet from the Stork format to the Kea format.
	subnet4, err := keaconfig.CreateSubnet4(1, lookupMock, mock)
	require.NoError(t, err)
	require.NotNil(t, *subnet4)

	// Make sure that the conversion was correct.
	require.Equal(t, "iterative", *subnet4.Allocator)
	require.True(t, *subnet4.Authoritative)
	require.Equal(t, "/tmp/bootfile", *subnet4.BootFileName)
	require.EqualValues(t, 1001, *subnet4.CacheMaxAge)
	require.EqualValues(t, 0.25, *subnet4.CacheThreshold)
	require.True(t, *subnet4.CalculateTeeTimes)
	require.Equal(t, "myclass", *subnet4.ClientClass)
	require.Equal(t, "example.com", *subnet4.DDNSGeneratedPrefix)
	require.True(t, *subnet4.DDNSOverrideClientUpdate)
	require.True(t, *subnet4.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *subnet4.DDNSQualifyingSuffix)
	require.Equal(t, "never", *subnet4.DDNSReplaceClientName)
	require.True(t, *subnet4.DDNSSendUpdates)
	require.True(t, *subnet4.DDNSUseConflictResolution)
	require.Equal(t, "foo-id", *subnet4.FourOverSixInterfaceID)
	require.Equal(t, "bar-id", *subnet4.FourOverSixInterface)
	require.Equal(t, "10.0.0.0/24", *subnet4.FourOverSixSubnet)
	require.Equal(t, "xyz", *subnet4.HostnameCharReplacement)
	require.Equal(t, "[A-z]", *subnet4.HostnameCharSet)
	require.EqualValues(t, 5, subnet4.ID)
	require.Equal(t, "etx0", *subnet4.Interface)
	require.True(t, *subnet4.MatchClientID)
	require.EqualValues(t, 1000, *subnet4.MaxValidLifetime)
	require.EqualValues(t, 500, *subnet4.MinValidLifetime)
	require.Equal(t, "192.0.2.1", *subnet4.NextServer)
	require.Len(t, subnet4.OptionData, 1)
	require.EqualValues(t, 5, subnet4.OptionData[0].Code)
	require.Equal(t, "dhcp4", subnet4.OptionData[0].Space)
	require.Len(t, subnet4.Pools, 1)
	require.Equal(t, "192.0.2.10-192.0.2.20", subnet4.Pools[0].Pool)
	require.Equal(t, "baz", subnet4.Pools[0].ClientClass)
	require.Len(t, subnet4.Pools[0].RequireClientClasses, 1)
	require.Len(t, subnet4.Pools[0].OptionData, 1)
	require.EqualValues(t, 6, subnet4.Pools[0].OptionData[0].Code)
	require.Equal(t, "dhcp4", subnet4.Pools[0].OptionData[0].Space)
	require.EqualValues(t, 300, *subnet4.RebindTimer)
	require.Len(t, subnet4.Relay.IPAddresses, 1)
	require.Equal(t, "10.0.0.1", subnet4.Relay.IPAddresses[0])
	require.EqualValues(t, 200, *subnet4.RenewTimer)
	require.Len(t, subnet4.RequireClientClasses, 1)
	require.Equal(t, "foo", subnet4.RequireClientClasses[0])
	require.Equal(t, "out-of-pool", *subnet4.ReservationMode)
	require.True(t, *subnet4.ReservationsGlobal)
	require.True(t, *subnet4.ReservationsInSubnet)
	require.True(t, *subnet4.ReservationsOutOfPool)
	require.Equal(t, "hostname.example.org", *subnet4.ServerHostname)
	require.True(t, *subnet4.StoreExtendedInfo)
	require.Equal(t, "192.0.2.0/24", subnet4.Subnet)
	require.EqualValues(t, 0.32, *subnet4.T1Percent)
	require.EqualValues(t, 0.44, *subnet4.T2Percent)
	require.EqualValues(t, 1001, *subnet4.ValidLifetime)
}

// Test converting an IPv6 subnet in Stork into the subnet configuration
// in Kea.
func TestCreateSubnet6(t *testing.T) {
	controller := gomock.NewController(t)

	// Mock a subnet in Stork.
	mock := NewMockSubnet(controller)
	poolMock := NewMockAddressPool(controller)
	pdPoolMock := NewMockPrefixPool(controller)

	// A mock to define an address pool.
	poolMock.EXPECT().GetLowerBound().AnyTimes().Return("2001:db8:1::10")
	poolMock.EXPECT().GetUpperBound().AnyTimes().Return("2001:db8:1::20")
	poolMock.EXPECT().GetKeaParameters().AnyTimes().Return(&keaconfig.PoolParameters{
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("baz"),
			RequireClientClasses: []string{"foo"},
		},
	})
	// Return DHCP options in the pool.
	poolMock.EXPECT().GetDHCPOptions().AnyTimes().Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        6,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp6",
		},
	})
	// A mock to define a delegated prefix pool.
	pdPoolMock.EXPECT().GetModel().AnyTimes().Return(&dhcpmodel.PrefixPool{
		Prefix:         "3001::/16",
		DelegatedLen:   64,
		ExcludedPrefix: "3001:1::/64",
	})
	pdPoolMock.EXPECT().GetKeaParameters().AnyTimes().Return(&keaconfig.PoolParameters{
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("baz"),
			RequireClientClasses: []string{"foo"},
		},
	})
	// Return DHCP options in the pool.
	pdPoolMock.EXPECT().GetDHCPOptions().AnyTimes().Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        7,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp6",
		},
	})
	// Subnet ID.
	mock.EXPECT().GetID(gomock.Any()).Return(int64(5))
	// Subnet prefix.
	mock.EXPECT().GetPrefix().Return("2001:db8:1::/64")
	// Return an address pool defined above.
	mock.EXPECT().GetAddressPools().Return([]dhcpmodel.AddressPoolAccessor{poolMock})
	// Return a delegated prefix pool defined above.
	mock.EXPECT().GetPrefixPools().Return([]dhcpmodel.PrefixPoolAccessor{pdPoolMock})
	// Return subnet-level Kea parameters.
	mock.EXPECT().GetKeaParameters(gomock.Eq(int64(1))).Return(&keaconfig.SubnetParameters{
		CacheParameters: keaconfig.CacheParameters{
			CacheMaxAge:    ptr[int64](1001),
			CacheThreshold: ptr[float32](0.25),
		},
		ClientClassParameters: keaconfig.ClientClassParameters{
			ClientClass:          ptr("myclass"),
			RequireClientClasses: []string{"foo"},
		},
		DDNSParameters: keaconfig.DDNSParameters{
			DDNSGeneratedPrefix:       ptr("example.com"),
			DDNSOverrideClientUpdate:  ptr(true),
			DDNSOverrideNoUpdate:      ptr(true),
			DDNSQualifyingSuffix:      ptr("example.org"),
			DDNSReplaceClientName:     ptr("never"),
			DDNSSendUpdates:           ptr(true),
			DDNSUseConflictResolution: ptr(true),
		},
		HostnameCharParameters: keaconfig.HostnameCharParameters{
			HostnameCharReplacement: ptr("xyz"),
			HostnameCharSet:         ptr("[A-z]"),
		},
		PreferredLifetimeParameters: keaconfig.PreferredLifetimeParameters{
			MaxPreferredLifetime: ptr[int64](800),
			MinPreferredLifetime: ptr[int64](300),
			PreferredLifetime:    ptr[int64](801),
		},
		ReservationParameters: keaconfig.ReservationParameters{
			ReservationMode:       ptr("out-of-pool"),
			ReservationsGlobal:    ptr(true),
			ReservationsInSubnet:  ptr(true),
			ReservationsOutOfPool: ptr(true),
		},
		TimerParameters: keaconfig.TimerParameters{
			CalculateTeeTimes: ptr(true),
			RebindTimer:       ptr[int64](300),
			RenewTimer:        ptr[int64](200),
			T1Percent:         ptr[float32](0.32),
			T2Percent:         ptr[float32](0.44),
		},
		ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
			MaxValidLifetime: ptr[int64](1000),
			MinValidLifetime: ptr[int64](500),
			ValidLifetime:    ptr[int64](1001),
		},
		Allocator:   ptr("iterative"),
		PDAllocator: ptr("random"),
		Interface:   ptr("etx0"),
		InterfaceID: ptr("id-foo"),
		RapidCommit: ptr(true),
		Relay: &keaconfig.Relay{
			IPAddresses: []string{"3000::1"},
		},
		ServerHostname:    ptr("hostname.example.org"),
		StoreExtendedInfo: ptr(true),
	})
	// Return subnet-level DHCP options.
	mock.EXPECT().GetDHCPOptions(gomock.Any()).Return([]dhcpmodel.DHCPOptionAccessor{
		keaconfig.DHCPOption{
			AlwaysSend:  true,
			Code:        5,
			Encapsulate: "foo",
			Fields:      []dhcpmodel.DHCPOptionFieldAccessor{},
			Space:       "dhcp6",
		},
	})
	// Do not return option definitions. This is not the area of the code
	// that we want to test here.
	lookupMock := NewMockDHCPOptionDefinitionLookup(controller)
	lookupMock.EXPECT().DefinitionExists(gomock.Any(), gomock.Any()).AnyTimes().Return(false)

	// Convert the subnet from the Stork format to the Kea format.
	subnet6, err := keaconfig.CreateSubnet6(1, lookupMock, mock)
	require.NoError(t, err)
	require.NotNil(t, subnet6)

	// Make sure that the conversion was correct.
	require.Equal(t, "iterative", *subnet6.Allocator)
	require.EqualValues(t, 1001, *subnet6.CacheMaxAge)
	require.EqualValues(t, 0.25, *subnet6.CacheThreshold)
	require.True(t, *subnet6.CalculateTeeTimes)
	require.Equal(t, "myclass", *subnet6.ClientClass)
	require.Equal(t, "example.com", *subnet6.DDNSGeneratedPrefix)
	require.True(t, *subnet6.DDNSOverrideClientUpdate)
	require.True(t, *subnet6.DDNSOverrideNoUpdate)
	require.Equal(t, "example.org", *subnet6.DDNSQualifyingSuffix)
	require.Equal(t, "never", *subnet6.DDNSReplaceClientName)
	require.True(t, *subnet6.DDNSSendUpdates)
	require.True(t, *subnet6.DDNSUseConflictResolution)
	require.Equal(t, "xyz", *subnet6.HostnameCharReplacement)
	require.Equal(t, "[A-z]", *subnet6.HostnameCharSet)
	require.EqualValues(t, 5, subnet6.ID)
	require.Equal(t, "etx0", *subnet6.Interface)
	require.EqualValues(t, 1000, *subnet6.MaxValidLifetime)
	require.EqualValues(t, 500, *subnet6.MinValidLifetime)
	require.Len(t, subnet6.OptionData, 1)
	require.EqualValues(t, 5, subnet6.OptionData[0].Code)
	require.Equal(t, "dhcp6", subnet6.OptionData[0].Space)
	require.Len(t, subnet6.Pools, 1)
	require.Equal(t, "2001:db8:1::10-2001:db8:1::20", subnet6.Pools[0].Pool)
	require.Equal(t, "baz", subnet6.Pools[0].ClientClass)
	require.Len(t, subnet6.Pools[0].RequireClientClasses, 1)
	require.Len(t, subnet6.Pools[0].OptionData, 1)
	require.EqualValues(t, 6, subnet6.Pools[0].OptionData[0].Code)
	require.Equal(t, "dhcp6", subnet6.Pools[0].OptionData[0].Space)
	require.Len(t, subnet6.PDPools, 1)
	require.Equal(t, "3001::", subnet6.PDPools[0].Prefix)
	require.EqualValues(t, 16, subnet6.PDPools[0].PrefixLen)
	require.Equal(t, "3001:1::", subnet6.PDPools[0].ExcludedPrefix)
	require.EqualValues(t, 64, subnet6.PDPools[0].ExcludedPrefixLen)
	require.Equal(t, "baz", subnet6.PDPools[0].ClientClass)
	require.Len(t, subnet6.PDPools[0].OptionData, 1)
	require.EqualValues(t, 7, subnet6.PDPools[0].OptionData[0].Code)
	require.Equal(t, "dhcp6", subnet6.PDPools[0].OptionData[0].Space)
	require.Len(t, subnet6.RequireClientClasses, 1)
	require.Equal(t, "foo", subnet6.RequireClientClasses[0])
	require.EqualValues(t, 300, *subnet6.RebindTimer)
	require.Len(t, subnet6.Relay.IPAddresses, 1)
	require.Equal(t, "3000::1", subnet6.Relay.IPAddresses[0])
	require.EqualValues(t, 200, *subnet6.RenewTimer)
	require.Len(t, subnet6.RequireClientClasses, 1)
	require.Equal(t, "foo", subnet6.RequireClientClasses[0])
	require.Equal(t, "out-of-pool", *subnet6.ReservationMode)
	require.True(t, *subnet6.ReservationsGlobal)
	require.True(t, *subnet6.ReservationsInSubnet)
	require.True(t, *subnet6.ReservationsOutOfPool)
	require.True(t, *subnet6.StoreExtendedInfo)
	require.Equal(t, "2001:db8:1::/64", subnet6.Subnet)
	require.EqualValues(t, 0.32, *subnet6.T1Percent)
	require.EqualValues(t, 0.44, *subnet6.T2Percent)
	require.EqualValues(t, 1001, *subnet6.ValidLifetime)
}
