package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that help for the stork-code-gen app returns valid switches.
func TestMainHelp(t *testing.T) {
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "std-option-defs")
	require.Contains(t, stdout, "--help")
	require.Contains(t, stdout, "--version")
}

// Test that help for the stork-code-gen std-option-defs returns valid
// switches.
func TestStdOptionDefsHelp(t *testing.T) {
	os.Args = make([]string, 5)
	os.Args[1] = "help"
	os.Args[2] = "std-option-defs"

	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, "--input")
	require.Contains(t, stdout, "--output")
	require.Contains(t, stdout, "--template")
}

// Test that the main function triggers generating option definitions.
func TestGenerateStdOptionDefs(t *testing.T) {
	// Create an input file with JSON contents.
	inputFile, err := os.CreateTemp(os.TempDir(), "code-gen-input-*.json")
	require.NoError(t, err)
	require.NotNil(t, inputFile)

	templateFile, err := os.CreateTemp(os.TempDir(), "code-gen-template-*.json")
	require.NoError(t, err)
	require.NotNil(t, templateFile)

	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
		templateFile.Close()
		os.Remove(templateFile.Name())
	}()

	_, err = inputFile.WriteString(`{
    "foo": "bar"
}`)
	require.NoError(t, err)

	_, err = templateFile.WriteString(`{{.foo}}`)
	require.NoError(t, err)

	// Prepare command line arguments generating a typescript code.
	os.Args = make([]string, 8)
	os.Args[1] = "std-option-defs"
	os.Args[2] = "--input"
	os.Args[3] = inputFile.Name()
	os.Args[4] = "--output"
	os.Args[5] = "stdout"
	os.Args[6] = "--template"
	os.Args[7] = templateFile.Name()

	// Run main function and capture output.
	stdoutBytes, _, err := testutil.CaptureOutput(main)
	require.NoError(t, err)
	stdout := string(stdoutBytes)
	require.Contains(t, stdout, `bar`)
}
