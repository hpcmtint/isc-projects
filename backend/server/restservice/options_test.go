package restservice

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/gen/models"
)

func TestFlattenDHCPOptions(t *testing.T) {
	restOptions := []*models.DHCPOption{
		{
			Code:        1001,
			Encapsulate: "option-1001",
			Options: []*models.DHCPOption{
				{
					Code:        1,
					Encapsulate: "option-1001.1",
				},
				{
					Code:        2,
					Encapsulate: "option-1001.2",
				},
			},
		},
		{
			Code:        1002,
			Encapsulate: "option-1002",
			Options: []*models.DHCPOption{
				{
					Code:        3,
					Encapsulate: "option-1002.3",
				},
				{
					Code:        4,
					Encapsulate: "option-1002.4",
				},
			},
		},
	}
	options := flattenDHCPOptions("dhcp4", restOptions)
	require.Len(t, options, 6)

	sort.Slice(options, func(i, j int) bool {
		return options[i].Code < options[j].Code
	})
	require.EqualValues(t, 1, options[0].Code)
	require.Equal(t, "option-1001", options[0].Space)
	require.Equal(t, "option-1001.1", options[0].Encapsulate)

	require.EqualValues(t, 2, options[1].Code)
	require.Equal(t, "option-1001", options[1].Space)
	require.Equal(t, "option-1001.2", options[1].Encapsulate)

	require.EqualValues(t, 3, options[2].Code)
	require.Equal(t, "option-1002", options[2].Space)
	require.Equal(t, "option-1002.3", options[2].Encapsulate)

	require.EqualValues(t, 4, options[3].Code)
	require.Equal(t, "option-1002", options[3].Space)
	require.Equal(t, "option-1002.4", options[3].Encapsulate)

	require.EqualValues(t, 1001, options[4].Code)
	require.Equal(t, "option-1001", options[4].Encapsulate)
	require.Equal(t, "dhcp4", options[4].Space)

	require.EqualValues(t, 1002, options[5].Code)
	require.Equal(t, "option-1002", options[5].Encapsulate)
	require.Equal(t, "dhcp4", options[5].Space)
}
