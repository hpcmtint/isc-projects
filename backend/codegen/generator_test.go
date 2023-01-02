package codegen

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

// Test that the generator is constructed properly.
func TestNewGenerator(t *testing.T) {
	// Act
	generator := NewGenerator()

	// Assert
	require.NotNil(t, generator)
	require.Nil(t, generator.data)
	require.Nil(t, generator.template)
}

// Test that the error is returned if the data file doesn't exist.
func TestReadDataFileForMissingFile(t *testing.T) {
	// Arrange
	generator := NewGenerator()

	// Act
	err := generator.ReadDataFile("/file/does/not/exist")

	// Assert
	require.Error(t, err)
}

// Test that the error is returned if the file isn't a JSON.
func TestReadDataFileForNonJSONFormat(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	tempFile, _ := os.CreateTemp("", "*")
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.WriteString("{")

	// Act
	err := generator.ReadDataFile(tempFile.Name())

	// Assert
	require.Error(t, err)
}

// Test that the valid JSON is loaded properly.
func TestReadDataFile(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	tempFile, _ := os.CreateTemp("", "*")
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.WriteString("{ }")

	// Act
	err := generator.ReadDataFile(tempFile.Name())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, generator.data)
}

// Test that the error is returned if the template file doesn't exist.
func TestReadTemplateFromFileForMissingFile(t *testing.T) {
	// Arrange
	generator := NewGenerator()

	// Act
	err := generator.ReadTemplateFile("/file/does/not/exist")

	// Assert
	require.Error(t, err)
}

// Test that the invalid template returns an error.
func TestReadTemplateFromFileForInvalidContent(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	tempFile, _ := os.CreateTemp("", "*")
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.WriteString("{{.")

	// Act
	err := generator.ReadTemplateFile(tempFile.Name())

	// Assert
	require.Error(t, err)
}

// Test that the valid template is loaded properly.
func TestReadTemplateFromFile(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	tempFile, _ := os.CreateTemp("", "*")
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.WriteString("{{.}}")

	// Act
	err := generator.ReadTemplateFile(tempFile.Name())

	// Assert
	require.NoError(t, err)
}

// Test that the output is generated properly.
func TestGenerate(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	generator.template, _ = template.New("foo").Parse("{{.Foo}}")
	generator.data = map[string]string{"Foo": "Bar"}
	var buffer bytes.Buffer

	// Act
	err := generator.Generate(&buffer)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "Bar", buffer.String())
}

// Test that the error is returned if the data don't correspond to the
// template.
func TestGenerateForIncompatibleData(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	generator.data = map[string]string{"Baz": "Bar"}
	var buffer bytes.Buffer

	tempFile, _ := os.CreateTemp("", "*")
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()
	tempFile.WriteString("{{.Foo}}")
	tempFile.Close()
	generator.ReadTemplateFile(tempFile.Name())

	// Act
	err := generator.Generate(&buffer)

	// Assert
	require.Error(t, err)
}

// Test that the output is generated properly and saved to a file.
func TestGenerateToFile(t *testing.T) {
	// Arrange
	generator := NewGenerator()
	generator.template, _ = template.New("foo").Parse("{{.Foo}}")
	generator.data = map[string]string{"Foo": "Bar"}

	tempFile, _ := os.CreateTemp("", "*")
	tempFile.Close()
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// Act
	err := generator.GenerateToFile(tempFile.Name())

	// Assert
	require.NoError(t, err)
	content, _ := ioutil.ReadFile(tempFile.Name())
	require.EqualValues(t, "Bar", content)
}
