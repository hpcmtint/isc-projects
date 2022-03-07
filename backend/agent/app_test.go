package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests creation of the BaseApp instance.
func TestNewBaseApp(t *testing.T) {
	app := NewBaseApp(AppTypeKea, makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234, true))
	require.NotNil(t, app)
	require.Equal(t, AppTypeKea, app.Type)
	require.Len(t, app.AccessPoints, 1)
	require.NotNil(t, app.configCaps)
}
