package agent

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that individual capabilities can be set for different
// daemons.
func TestSetCaps(t *testing.T) {
	caps := newConfigCaps()
	require.NotNil(t, caps)

	caps.set("dhcp4", "host_cmds")
	caps.set("dhcp4", "lease_cmds")
	caps.set("dhcp6", "subnet_cmds")

	// Convert to transmittable format.
	converted := caps.convertToProtobuf()
	require.Len(t, converted, 2)

	// Ensure that the capabilities are sorted in the predictable order.
	sort.Slice(converted, func(i, j int) bool { return converted[i].Daemon < converted[j].Daemon })
	sort.Slice(converted[0].Caps, func(i, j int) bool { return converted[0].Caps[i].Name < converted[0].Caps[j].Name })

	// There should be capabilities for two daemons.
	require.Equal(t, "dhcp4", converted[0].Daemon)
	require.Equal(t, "dhcp6", converted[1].Daemon)

	require.Len(t, converted[0].Caps, 2)
	require.Equal(t, "host_cmds", converted[0].Caps[0].Name)
	require.Equal(t, "lease_cmds", converted[0].Caps[1].Name)

	require.Len(t, converted[1].Caps, 1)
	require.Equal(t, "subnet_cmds", converted[1].Caps[0].Name)

	// Unset some capabilities.
	caps.unset("dhcp4", "host_cmds")
	caps.unset("dhcp6", "subnet_cmds")

	converted = caps.convertToProtobuf()
	// Capabilities for the dhcp6 daemon should no longer be present.
	require.Len(t, converted, 1)
	require.Equal(t, "dhcp4", converted[0].Daemon)
	require.Len(t, converted[0].Caps, 1)
	require.Equal(t, "lease_cmds", converted[0].Caps[0].Name)
}

// Test clearing all capabilities and clearing the capabilities for
// individual daemons.
func TestClearCaps(t *testing.T) {
	caps := newConfigCaps()
	require.NotNil(t, caps)

	// Set capabilities for three daemons.
	caps.set("dhcp4", "host_cmds")
	caps.set("dhcp4", "lease_cmds")
	caps.set("dhcp6", "subnet_cmds")
	caps.set("ca", "log_cmds")

	// Clear all capabilities for the dhcp4 daemon.
	caps.clear("dhcp4")

	// Output all capabilities.
	converted := caps.convertToProtobuf()
	require.Len(t, converted, 2)

	// Ensure predictable sort order.
	sort.Slice(converted, func(i, j int) bool { return converted[i].Daemon < converted[j].Daemon })

	// We should have the capabilities set for ca and dhcp6 daemons
	// but no capabilities for the dhcp4 daemon.
	require.Equal(t, "ca", converted[0].Daemon)
	require.Equal(t, "dhcp6", converted[1].Daemon)

	require.Len(t, converted[0].Caps, 1)
	require.Equal(t, converted[0].Caps[0].Name, "log_cmds")
	require.Len(t, converted[1].Caps, 1)
	require.Equal(t, converted[1].Caps[0].Name, "subnet_cmds")

	// Remove all remaining capabilities.
	caps.clearAll()

	// There should be none.
	converted = caps.convertToProtobuf()
	require.Empty(t, converted)
}
