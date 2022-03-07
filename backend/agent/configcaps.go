package agent

import (
	agentapi "isc.org/stork/api"
)

// A structure describing configuration capabilities of an app.
type configCaps struct {
	// The first map's key is a daemon name to which the capabilities
	// apply. The second map (with a bool value) indicates if the
	// capability is present (when true) or absent (when false or
	// when the entry doesn't exist).
	caps map[string]map[string]bool
}

// Creates empty capabilities instance.
func newConfigCaps() *configCaps {
	return &configCaps{
		caps: make(map[string]map[string]bool),
	}
}

// Clears all capabilities.
func (c configCaps) clearAll() {
	for daemon := range c.caps {
		c.clear(daemon)
	}
}

// Clears all capabilities for a daemon.
func (c configCaps) clear(daemon string) {
	delete(c.caps, daemon)
}

// Sets new capability for a daemon.
func (c configCaps) set(daemon, cap string) {
	if _, ok := c.caps[daemon]; !ok {
		c.caps[daemon] = make(map[string]bool)
	}
	c.caps[daemon][cap] = true
}

// Unsets specified capability for a daemon if it exists.
func (c configCaps) unset(daemon, cap string) {
	if _, ok := c.caps[daemon]; !ok {
		return
	}
	c.caps[daemon][cap] = false
}

// Outputs all capabilities in a format directly transmittable over gRPC.
func (c configCaps) convertToProtobuf() (converted []*agentapi.DaemonConfigCaps) {
	for daemon, caps := range c.caps {
		d := daemon
		pcaps := &agentapi.DaemonConfigCaps{
			Daemon: d,
		}
		for cap, set := range caps {
			if set {
				c := cap
				pcaps.Caps = append(pcaps.Caps, &agentapi.ConfigCap{
					Name: c,
				})
			}
		}
		if len(pcaps.Caps) > 0 {
			converted = append(converted, pcaps)
		}
	}
	return
}
