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

package printer

import (
	"github.com/pkg/errors"

	"github.com/McKael/ghreleasechecker/gh"
)

// Options contains printer-specific options
type Options map[string]interface{}

// Printer is an interface used to print objects.
type Printer interface {
	// PrintReleases receives a list of releases,
	// formats it and prints it to stdout.
	PrintReleases([]gh.Release) error
}

// NewPrinter returns a printer of the requested kind
func NewPrinter(printerName string, o Options) (Printer, error) {
	switch printerName {
	case "plain", "":
		return NewPrinterPlain(o)
	case "json":
		return NewPrinterJSON(o)
	case "yaml":
		return NewPrinterYAML(o)
	case "template":
		return NewPrinterTemplate(o)
	}
	return nil, errors.New("unknown printer")
}
