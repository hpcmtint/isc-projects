package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/jessevdk/go-flags"
	"isc.org/stork/server/database/migrations"
	"os"
)

type cmdOpts struct {
}

type upOpts struct {
	Target string `short:"t" long:"target" description:"Target database schema version"`
}

type Opts struct{
	DatabaseName string `short:"d" long:"database" description:"database name" required:"true"`
	UserName string `short:"u" long:"user" description:"database user name" required:"true"`
	Init cmdOpts `command:"init" description:"Create schema versioning table in the database"`
	Up upOpts `command:"up" description:"Run all available migrations"`
	Down cmdOpts `command:"down" description:"Revert last migration"`
	Reset cmdOpts `command:"reset" description:"Revert all migrations"`
	Version cmdOpts `command:"version" description:"Print current migration version"`
	SetVersion cmdOpts `command:"set_version" description:"Set database version without running migrations"`
}

func main() {
	opts := Opts{}
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Prompt the user for database password.
	fmt.Printf("database password: ")
	password, err := terminal.ReadPassword(0)
	fmt.Printf("\n")

	if err != nil {
		exitf(nil, err.Error())
	}

	var args []string
	args = append(args, parser.Active.Name)
	if parser.Active.Name == "up" && len(opts.Up.Target) > 0 {
		args = append(args, opts.Up.Target)
	}

	// Use the provided credentials to connect to the database.
	oldVersion, newVersion, err := dbmigs.Migrate(&dbmigs.DbConnOptions{
		User:     opts.UserName,
		Password: string(password),
		Database: opts.DatabaseName,
	}, args...)

	if err != nil {
		exitf(nil, err.Error())
	}

	if newVersion != oldVersion {
		fmt.Printf("Migrated database from version %d to %d\n", oldVersion, newVersion)
	} else {
		fmt.Printf("Database version is %d\n", oldVersion)
	}
}

// Prints error string to stderr.
func errorf(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
}

// Prints error string to stderr and exists with exit code 1.
// If the usagefn is not nil it is invoked to present program
// usage information.
func exitf(usagefn func(), s string, args ...interface{}) {
	errorf(s, args...)
	if usagefn != nil {
		usagefn()
	}
	os.Exit(1)
}
