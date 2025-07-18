// Copyright © 2018 Mikael Berthe <mikael@lilotux.net>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/McKael/ghreleasechecker/gh"
	"github.com/McKael/ghreleasechecker/gh/printer"
)

// AppName is the CLI application name
const AppName = "ghreleasechecker"

// Version is the CLI application version
var Version = "0.0.6-dev"

// Command line parameters
var (
	debug     bool
	cfgFile   string
	token     string
	showBody  bool
	output    string
	template  string
	colorMode string
	readOnly  bool
	wait      bool
	version   bool
)

var ghConfig *gh.Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   AppName,
	Short: "A release watcher for Github projects",
	Long: `ghReleaseChecker is a release watcher for Github projects.

It will display projects with a new release since the last time it was run.

The list of repositories to be checked is provided in a YAML configuration
file.  A file (in JSON format) is used to keep state between calls; this
file path must be set in the configuration file.

Note that ghReleaseChecker uses the Github official API, which is rate-limited
(especially for anonymous users).  One can use a Github token to increase the
rate.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			fmt.Printf("This is %s version %s.\n", AppName, Version)
			os.Exit(0)
		}

		// The configuration should have been loaded already
		if ghConfig == nil {
			fmt.Fprintln(os.Stderr, "Internal error: no configuration loaded")
			os.Exit(1)
		}

		releases, err := ghConfig.CheckReleases(readOnly)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		displayReleases(releases)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags and configuration settings.
	RootCmd.PersistentFlags().BoolVar(&version, "version", false, "Display version")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Display debugging details")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/"+AppName+"/"+AppName+".yaml)")
	RootCmd.PersistentFlags().StringVarP(&token, "token", "t", "",
		"Github API user token")
	RootCmd.PersistentFlags().BoolVar(&wait, "wait", false, "Wait when rate limit is exceeded")

	RootCmd.Flags().StringVarP(&output, "output", "o", "", "Output handler (default: plain)")
	RootCmd.Flags().StringVar(&template, "template", "", "Go template (for output=template)")
	RootCmd.Flags().StringVar(&colorMode, "color", "", "Color mode (auto|on|off; for output=template)")
	RootCmd.Flags().BoolVar(&showBody, "show-body", false, "Display release body (for output=plain)")
	RootCmd.Flags().BoolVar(&readOnly, "read-only", false, "Do not update the state file")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cfgFile = filepath.Join(home, ".config", AppName, AppName+".yaml")
	}

	// Read config file.
	var err error
	ghConfig, err = gh.ReadConfig(cfgFile, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load file '%s': %s\n", cfgFile, err)
		os.Exit(1)
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Default printer options from configuration file
	if ghConfig.Printer != nil {
		var do, cm, tp *string
		var sb *bool
		do = ghConfig.Printer.DefaultOutput
		if ghConfig.Printer.PlainPrinter != nil {
			sb = ghConfig.Printer.PlainPrinter.ShowBody
			cm = ghConfig.Printer.TemplatePrinter.ColorMode
		}
		if ghConfig.Printer.TemplatePrinter != nil {
			tp = ghConfig.Printer.TemplatePrinter.Template
		}

		// Use values from configuration file when the options are
		// not provided on the command line.
		fl := RootCmd.Flags()
		if !fl.Lookup("show-body").Changed && sb != nil {
			showBody = *sb
		}
		if !fl.Lookup("color").Changed && cm != nil {
			colorMode = *cm
			switch colorMode {
			// "true" or "false" can be set by YAML parser!
			case "true":
				colorMode = "on"
			case "false":
				colorMode = "off"
			}
		}

		outputFromCLI := fl.Lookup("output").Changed
		templateFromCLI := fl.Lookup("template").Changed

		if !outputFromCLI {
			if templateFromCLI {
				output = "template"
			} else if do != nil {
				output = *do
			}
		}
		if !templateFromCLI && tp != nil && (!outputFromCLI || output == "template") {
			template = *tp
		}
		if output == "template" && template == "" && !outputFromCLI {
			logrus.Error("Cannot use template output (no template)")
			// Fall back to default (plain) output
			output = ""
		}
	}

	if RootCmd.PersistentFlags().Lookup("wait").Changed {
		ghConfig.Wait = wait // Overwrite config file value
	}
}

func displayReleases(rr []gh.ReleaseList) {
	opt := make(printer.Options)

	switch output {
	case "", "plain":
		if showBody {
			opt["show_body"] = true
		}
	case "template":
		opt["template"] = template
		if colorMode != "" {
			opt["color_mode"] = colorMode
		}
	}

	p, err := printer.NewPrinter(output, opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not initialize printer: %s\n", err)
		os.Exit(1)
	}

	if err = p.PrintReleases(rr); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not display releases: %s\n", err)
		os.Exit(1)
	}

	return
}
