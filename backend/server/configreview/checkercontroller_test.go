package configreview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the checker controller is constructed properly.
func TestNewCheckerController(t *testing.T) {
	// Act
	controller := newCheckerController()

	// Assert
	require.NotNil(t, controller)
}

// Test that the checker state is returned properly.
func TestGetGlobalState(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.SetGlobalState("foo", CheckerStateEnabled)
	controller.SetGlobalState("bar", CheckerStateDisabled)

	// Act
	foo := controller.GetGlobalState("foo")
	bar := controller.GetGlobalState("bar")
	baz := controller.GetGlobalState("baz")

	// Assert
	require.True(t, foo)
	require.False(t, bar)
	require.True(t, baz)
}

// Test that the checker state is set properly.
func TestSetGlobalState(t *testing.T) {
	// Arrange
	controller := newCheckerController()

	// Act
	controller.SetGlobalState("foo", CheckerStateEnabled)
	controller.SetGlobalState("bar", CheckerStateDisabled)
	controller.SetGlobalState("baz", CheckerStateInherit)

	// Assert
	require.True(t, controller.IsCheckerEnabledForDaemon(0, "foo"))
	require.False(t, controller.IsCheckerEnabledForDaemon(0, "bar"))
	require.True(t, controller.IsCheckerEnabledForDaemon(0, "baz"))
}

// Test that the checker state for a specific daemon is set properly.
func TestSetStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()

	// Act
	controller.SetStateForDaemon(1, "foo", CheckerStateEnabled)
	controller.SetStateForDaemon(2, "bar", CheckerStateDisabled)
	controller.SetStateForDaemon(3, "baz", CheckerStateInherit)

	// Assert
	require.True(t, controller.IsCheckerEnabledForDaemon(1, "foo"))
	require.False(t, controller.IsCheckerEnabledForDaemon(2, "bar"))
	require.True(t, controller.IsCheckerEnabledForDaemon(3, "baz"))
}

// Test that the checker state is correctly inherited for a specific daemon.
func TestSetInheritedStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.SetGlobalState("foo", CheckerStateEnabled)
	controller.SetGlobalState("bar", CheckerStateDisabled)
	controller.SetGlobalState("baz", CheckerStateDisabled)

	// Act
	controller.SetStateForDaemon(1, "foo", CheckerStateInherit)
	controller.SetStateForDaemon(2, "bar", CheckerStateInherit)
	controller.SetStateForDaemon(3, "baz", CheckerStateInherit)
	controller.SetGlobalState("baz", CheckerStateEnabled)

	// Assert
	require.True(t, controller.IsCheckerEnabledForDaemon(1, "foo"))
	require.False(t, controller.IsCheckerEnabledForDaemon(2, "bar"))
	require.True(t, controller.IsCheckerEnabledForDaemon(3, "baz"))
}

// Test that the checker states are merged properly.
func TestIsCheckerEnabledForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.SetGlobalState("foo", CheckerStateEnabled)
	controller.SetGlobalState("fee", CheckerStateDisabled)
	controller.SetGlobalState("bar", CheckerStateEnabled)
	controller.SetStateForDaemon(1, "bar", CheckerStateEnabled)
	controller.SetGlobalState("baz", CheckerStateEnabled)
	controller.SetStateForDaemon(1, "baz", CheckerStateDisabled)
	controller.SetGlobalState("biz", CheckerStateDisabled)
	controller.SetStateForDaemon(1, "biz", CheckerStateEnabled)
	controller.SetGlobalState("boz", CheckerStateDisabled)
	controller.SetStateForDaemon(1, "boz", CheckerStateDisabled)

	// Act
	foo := controller.IsCheckerEnabledForDaemon(1, "foo")
	fee := controller.IsCheckerEnabledForDaemon(1, "fee")
	bar := controller.IsCheckerEnabledForDaemon(1, "bar")
	baz := controller.IsCheckerEnabledForDaemon(1, "baz")
	biz := controller.IsCheckerEnabledForDaemon(1, "biz")
	boz := controller.IsCheckerEnabledForDaemon(1, "boz")

	// Assert
	require.True(t, foo)
	require.False(t, fee)
	require.True(t, bar)
	require.False(t, baz)
	require.True(t, biz)
	require.False(t, boz)
}

// Test that own state of config checker is returned properly.
func TestGetCheckerOwnStateForDaemon(t *testing.T) {
	// Arrange
	controller := newCheckerController()
	controller.SetStateForDaemon(1, "foo", CheckerStateDisabled)
	controller.SetStateForDaemon(1, "bar", CheckerStateEnabled)
	controller.SetStateForDaemon(1, "baz", CheckerStateInherit)

	// Act
	foo := controller.GetCheckerOwnState(1, "foo")
	bar := controller.GetCheckerOwnState(1, "bar")
	baz := controller.GetCheckerOwnState(1, "baz")
	boz := controller.GetCheckerOwnState(1, "boz")

	// Assert
	require.EqualValues(t, CheckerStateDisabled, foo)
	require.EqualValues(t, CheckerStateEnabled, bar)
	require.EqualValues(t, CheckerStateInherit, baz)
	require.EqualValues(t, CheckerStateInherit, boz)
}

// Test that the config checker state is serialized to string properly.
func TestConfigCheckerStateToString(t *testing.T) {
	require.EqualValues(t, "disabled", CheckerStateDisabled.String())
	require.EqualValues(t, "enabled", CheckerStateEnabled.String())
	require.EqualValues(t, "inherit", CheckerStateInherit.String())
	require.EqualValues(t, "unknown", CheckerState(42).String())
}