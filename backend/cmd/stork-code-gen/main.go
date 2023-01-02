package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"isc.org/stork"
	"isc.org/stork/codegen"
)

// Generates a structure with option definitions in a selected programming
// language.
func generateStdOptionDefs(c *cli.Context) error {
	input := c.String("input")
	output := c.String("output")
	template := c.String("template")

	generator := codegen.NewGenerator()

	err := generator.ReadDataFile(input)
	if err != nil {
		return err
	}

	err = generator.ReadTemplateFile(template)
	if err != nil {
		return err
	}

	if output == "stdout" {
		err = generator.Generate(os.Stdout)
	} else {
		err = generator.GenerateToFile(output)
	}

	return err
}

// Man function exposing command line parameters.
func main() {
	app := &cli.App{
		Name:     "Stork Code Gen",
		Usage:    "Code generator used in Stork development",
		Version:  stork.Version,
		HelpName: "stork-code-gen",
		Commands: []*cli.Command{
			{
				Name:      "std-option-defs",
				Usage:     "Generate standard option definitions from JSON spec.",
				UsageText: "stork-code-gen std-option-defs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "input",
						Usage:    "Path to the input file holding option definitions' specification.",
						Required: true,
						Aliases:  []string{"i"},
					},
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Path to the output file or 'stdout' to print the generated code in the terminal.",
						Required: true,
						Aliases:  []string{"o"},
					},
					&cli.StringFlag{
						Name:    "template",
						Usage:   "Path to the template file used to generate the output file. The generated code is embedded in the template file.",
						Aliases: []string{"t"},
					},
				},
				Action: generateStdOptionDefs,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
