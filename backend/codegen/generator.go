package codegen

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"text/template"

	"github.com/pkg/errors"
)

// Generates a file based on a template.
type Generator struct {
	data     any
	template *template.Template
}

// Constructs the generator instance.
func NewGenerator() *Generator {
	return &Generator{}
}

// Reads the data used to fill the template. The provided file must be in JSON
// format.
func (g *Generator) ReadDataFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "cannot open file (%s)", path)
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "cannot read from file (%s)", path)
	}

	err = json.Unmarshal(bytes, &g.data)
	return errors.Wrapf(err, "cannot parse JSON data from file (%s)", path)
}

// Reads the template from a file.
func (g *Generator) ReadTemplateFile(templatePath string) (err error) {
	g.template, err = template.
		New(path.Base(templatePath)).
		Option("missingkey=error").
		ParseFiles(templatePath)
	return errors.Wrapf(err, "cannot parse the template from file (%s)", templatePath)
}

// Generates a file by filling the template with data.
func (g *Generator) GenerateToFile(outputPath string) (err error) {
	if g.data == nil {
		return errors.Errorf("the data are empty")
	}

	if g.template == nil {
		return errors.Errorf("template is not set")
	}

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return errors.Wrapf(err, "cannot open file (%s) to write", outputPath)
	}
	defer f.Close()
	return g.Generate(f)
}

// Generates a file content by filling the template with data.
func (g *Generator) Generate(writer io.Writer) error {
	return g.template.Execute(writer, g.data)
}
