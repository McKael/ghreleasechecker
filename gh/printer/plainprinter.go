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
	"strings"

	"github.com/kr/text"

	"github.com/McKael/ghreleasechecker/gh"
)

// PlainPrinter is a plain text printer (the default one)
type PlainPrinter struct {
	showBody bool
}

// NewPrinterPlain returns a plaintext printer
func NewPrinterPlain(options Options) (*PlainPrinter, error) {
	p := &PlainPrinter{}
	if body, ok := options["show_body"]; ok {
		p.showBody = body.(bool)
	}
	return p, nil
}

// PrintReleases displays a list of releases to the standard output
func (p *PlainPrinter) PrintReleases(rr []gh.Release) error {
	for _, r := range rr {
		pre := ""
		if r.PreRelease != nil && *r.PreRelease {
			pre = "pre-"
		}
		fmt.Printf("New %srelease for %s: %s\n", pre, r.Repo, r.Version)
		if r.Tag != nil {
			fmt.Printf("  Tag: %s\n", *r.Tag)
		}
		if r.PublishDate != nil {
			fmt.Printf("  Date: %s\n", r.PublishDate.Local().
				Format("2006-01-02 15:04:05 -0700 MST"))
		}

		if r.Body != nil && p.showBody {
			fmt.Println("  Release body:")
			fmt.Println(text.Indent(strings.TrimSpace(*r.Body), "    "))
		}

		fmt.Println()
	}

	return nil
}
