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

package gh

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Config contains the utility configuration details
type Config struct {
	// Github-related configuration items
	Token        *string `json:"token"` // Github token, optional
	StateFile    string  `json:"state_file"`
	Repositories []struct {
		Repo        string `json:"repo"`        // owner/repo_name
		Prereleases *bool  `json:"prereleases"` // include prereleases
	} `json:"repositories"`

	// Printer is optional and contains the default configuration for
	// the different printers (plaintext, template...).
	Printer *struct {
		DefaultOutput *string `json:"default_output"`

		PlainPrinter *struct {
			ShowBody *bool `json:"show_body"`
		} `json:"plain_printer"`
		TemplatePrinter *struct {
			Template  *string `json:"template"`
			ColorMode *string `json:"color_mode"`
		} `json:"template_printer"`
	}

	// Private objects
	states *States
	client *github.Client
}

// States is a struct that contains the states of all checked releases
type States struct {
	Repositories map[string]RepoState `json:"repositories"`
}

// RepoState contains the state of a given repository
type RepoState struct {
	Repo        string            `json:"repo"`
	Version     string            `json:"version"`
	Tag         *string           `json:"tag,omitempty"`
	PreRelease  *bool             `json:"prerelease,omitempty"`
	PublishDate *github.Timestamp `json:"publish_date,omitempty"`

	body *string
}

// ReadConfig reads an YAML file containing the configuration
// It returns the configuration details, or an error.
func ReadConfig(filePath string) (*Config, error) {

	confdata, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read configuration file")
	}

	var c Config

	if err := yaml.Unmarshal(confdata, &c); err != nil {
		return nil, errors.Wrap(err, "cannot parse configuration file")
	}

	var tc *http.Client
	if c.Token != nil {
		tc = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *c.Token},
		))
	}
	c.client = github.NewClient(tc)

	return &c, nil
}

// loadStateFile reads a JSON file containing the state of previous queries
func (c *Config) loadStateFile() error {
	if c == nil {
		return errors.New("internal error: Config not set")
	}

	if c.StateFile == "" {
		// We don't use a state file
		return nil
	}

	data, err := ioutil.ReadFile(c.StateFile)
	if err != nil {
		// return errors.Wrap(err, "cannot read state file")

		// The file might not exist
		return nil
	}

	var s States

	if err := json.Unmarshal(data, &s); err != nil {
		return errors.Wrap(err, "cannot parse JSON state file")
	}

	c.states = &s

	return nil
}

// writeStateFile writes a JSON file containing the state of previous queries
// Note: It is not very safe; data can be lost on storage failure (e.g. on disk
// full condition).
func (c *Config) writeStateFile() error {
	if c == nil {
		return errors.New("internal error: Config not set")
	}

	if c.StateFile == "" {
		// We don't use a state file
		return nil
	}

	data, err := json.Marshal(c.states)
	if err != nil {
		return errors.Wrap(err, "failed to JSON-encode states")
	}

	if err := ioutil.WriteFile(c.StateFile, data, 0600); err != nil {
		return errors.Wrap(err, "cannot write state file") // XXX
	}

	return nil
}
