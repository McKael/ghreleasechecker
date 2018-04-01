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
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Release contains a repository release
type Release struct {
	*RepoState
	Body *string `json:"body"`
}

// CheckReleases checks all configured repositories for new releases
func (c *Config) CheckReleases() ([]Release, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("uninitialized client")
	}

	if err := c.loadStateFile(); err != nil {
		return nil, errors.Wrap(err, "cannot load state file")
	}

	var workCount int
	newReleases := make(chan *Release)
	ctx := context.Background()

	for _, r := range c.Repositories {
		workCount++
		r := r
		// XXX That's not really good, we should limit the number of
		// concurrent queries.
		go func(newRel chan<- *Release) {
			ost := c.getOldState(r.Repo)
			nr, err := checkRepoReleases(ctx, c.client, r.Prereleases, ost)
			if err != nil {
				logrus.Errorf("Check for repo '%s' failed: %s\n", r.Repo, err)
				newRel <- nil
				return
			}
			newRel <- nr
		}(newReleases)
	}

	var newReleaseList []Release
	for workCount > 0 {
		rel := <-newReleases
		workCount--

		if rel == nil {
			continue
		}

		// Queue the release for states updates
		newReleaseList = append(newReleaseList, *rel)
	}

	var changed bool
	for _, s := range newReleaseList {
		// Update states
		if c.states == nil {
			rm := make(map[string]RepoState)
			c.states = &States{Repositories: rm}
		}
		c.states.Repositories[s.Repo] = *s.RepoState
		changed = true
	}

	// Save states if needed
	if changed {
		logrus.Debug("States needs saving...")
		if err := c.writeStateFile(); err != nil {
			return newReleaseList, errors.Wrap(err, "cannot write state file")
		}
	}

	return newReleaseList, nil
}

func (c *Config) getOldState(repo string) RepoState {
	if c.states != nil && c.states.Repositories != nil {
		if r, ok := c.states.Repositories[repo]; ok {
			return r
		}
	}
	return RepoState{Repo: repo}
}

func checkRepoReleases(ctx context.Context, client *github.Client, prereleases *bool, prevState RepoState) (*Release, error) {
	pp := strings.Split(prevState.Repo, "/")
	if len(pp) != 2 {
		return nil, errors.Errorf("invalid repository name '%s'", prevState.Repo)
	}

	//logrus.Debugf("Project '%s'", prevState.Repo)
	logrus.Debugf("[%s] Previous version: '%s'", prevState.Repo, prevState.Version)
	/*
		if prevState.Tag != nil {
			logrus.Debugf(" [%s] Previous tag: '%s'", prevState.Repo, *prevState.Tag)
		}
	*/

	rr, _, err := client.Repositories.ListReleases(ctx, pp[0], pp[1], nil)
	if err != nil {
		return nil, errors.Wrap(err, "client.Repositories.ListReleases() failed")
	}

	lastCheck := false

	for _, r := range rr {
		if lastCheck {
			break
		}

		if r.GetDraft() {
			continue // Skip drafts
		}

		newVersion := r.GetName()

		if prevState.Version == newVersion {
			// We have already seen this release,
			// let's skip everything from here.
			// We'll still check this one though...
			lastCheck = true
		}

		if r.Prerelease != nil && *r.Prerelease {
			// This is a pre-release
			if prereleases == nil || !*prereleases {
				continue
			}
		}

		newTag := r.GetTagName()
		newDate := r.GetPublishedAt()

		if (prevState.Version != newVersion) ||
			(prevState.PublishDate == nil || prevState.PublishDate.Unix() < newDate.Unix()) ||
			(prevState.Tag == nil || *prevState.Tag != newTag) {
			if prevState.Version == newVersion && newVersion != "" {
				logrus.Infof("[%s] Same version but date or tag has changed", prevState.Repo)
			}
			rel := &Release{
				RepoState: &RepoState{
					Repo:        prevState.Repo,
					Version:     newVersion,
					Tag:         r.TagName,
					PreRelease:  r.Prerelease,
					PublishDate: r.PublishedAt,
					body:        r.Body,
				},
				Body: r.Body,
			}
			return rel, nil
		}
	}

	return nil, nil
}
