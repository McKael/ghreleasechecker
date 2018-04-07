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
	"fmt"

	"github.com/ghodss/yaml"

	"github.com/McKael/ghreleasechecker/gh"
)

// YAMLPrinter is a YAML printer (the default one)
type YAMLPrinter struct {
}

// NewPrinterYAML returns a YAML printer
func NewPrinterYAML(options Options) (*YAMLPrinter, error) {
	p := &YAMLPrinter{}
	return p, nil
}

// PrintReleases displays a list of releases in YAML format
func (p *YAMLPrinter) PrintReleases(rr []gh.ReleaseList) error {
	if len(rr) == 0 {
		return nil
	}
	bb, err := yaml.Marshal(rr)
	if err != nil {
		return err
	}
	fmt.Println(string(bb))
	return nil
}
