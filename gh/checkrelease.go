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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// releaseWorkerCount is the number of worker goroutines to query the
// Github API.
const releaseWorkerCount = 3

// Release contains a repository release
type Release struct {
	*RepoState
	Body *string `json:"body"`
}

// ReleaseList represents a list of new releases for a given project
type ReleaseList []*Release

// checkReleaseWorker is a worker to check new releases
func (c *Config) checkReleaseWorker(ctx context.Context, wID int, repoQueue <-chan RepoConfig, newRel chan<- ReleaseList) {
	logrus.Debugf("[%d] checkReleaseWorker starting.", wID)
	for r := range repoQueue {
		logrus.Debugf("[%d] checkReleaseWorker - repository '%s'", wID, r.Repo)
		ost := c.getOldState(r.Repo)
		nr, err := c.checkRepoReleases(ctx, wID, r.Prereleases, ost)
		if err != nil {
			logrus.Errorf("[%d] Check for repo '%s' failed: %s\n", wID, r.Repo, err)
			newRel <- nil
			continue
		}
		newRel <- nr
		logrus.Debugf("[%d] checkReleaseWorker - job done.", wID)
	}
	logrus.Debugf("[%d] checkReleaseWorker leaving.", wID)
}

// CheckReleases checks all configured repositories for new releases
func (c *Config) CheckReleases(readOnly bool) ([]ReleaseList, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("uninitialized client")
	}

	if err := c.loadStateFile(); err != nil {
		return nil, errors.Wrap(err, "cannot load state file")
	}

	newReleases := make(chan ReleaseList)
	repoQ := make(chan RepoConfig)
	ctx := context.Background()

	// Launch workers
	for i := 0; i < releaseWorkerCount; i++ {
		go c.checkReleaseWorker(ctx, i+1, repoQ, newReleases)
	}

	// Queue jobs
	go func() {
		for _, r := range c.Repositories {
			repoQ <- r
		}
		close(repoQ)
	}()

	// Collect results
	var newReleaseList []ReleaseList
	for resultCount := len(c.Repositories); resultCount > 0; {
		rel := <-newReleases
		resultCount--

		if len(rel) == 0 {
			continue
		}

		// Queue the release for states updates
		newReleaseList = append(newReleaseList, rel)
	}

	// Leave now if the result list is empty or if we don't need to save them
	if len(newReleaseList) == 0 || readOnly {
		return newReleaseList, nil
	}

	// Update repository states
	for _, s := range newReleaseList {
		// Update states
		if c.states == nil {
			rm := make(map[string]RepoState)
			c.states = &States{Repositories: rm}
		}
		c.states.Repositories[s[0].Repo] = *(s[0].RepoState)
	}

	// Save states
	logrus.Debug("Saving states...")
	if err := c.writeStateFile(); err != nil {
		return newReleaseList, errors.Wrap(err, "cannot write state file")
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

func (c *Config) checkRepoReleases(ctx context.Context, wID int, prereleases bool, prevState RepoState) (ReleaseList, error) {
	client := c.client

	pp := strings.Split(prevState.Repo, "/")
	if len(pp) != 2 {
		return nil, errors.Errorf("invalid repository name '%s'", prevState.Repo)
	}

	//logrus.Debugf("[%d] Project '%s'", wID, prevState.Repo)
	logrus.Debugf("[%d] Repository '%s' - Previous version: '%s'", wID, prevState.Repo, prevState.Version)
	/*
		if prevState.Tag != nil {
			logrus.Debugf("[%d]  Previous tag: '%s'", wID, *prevState.Tag)
		}
	*/

	rr, resp, err := client.Repositories.ListReleases(ctx, pp[0], pp[1], nil)
	if err != nil {
		if resp != nil && resp.Response != nil &&
			resp.Response.StatusCode == 403 && resp.Remaining == 0 {
			logrus.Infof("[%d] We're being rate-limited.  Limit reset at %v", wID, resp.Reset)

			if c.Wait {
				d := resp.Reset.Sub(time.Now())
				if d < 0 {
					d = 0
				}
				d += 30 * time.Second
				d -= d % time.Second

				// Let's wait and try again...
				logrus.Infof("[%d] Waiting for %v", wID, d)
				time.Sleep(d)
				return c.checkRepoReleases(ctx, wID, prereleases, prevState)
			}
		}
		return nil, errors.Wrap(err, "client.Repositories.ListReleases() failed")
	}

	lastCheck := false
	var newReleaseList ReleaseList

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
			if !prereleases {
				continue
			}
		}

		newTag := r.GetTagName()
		newDate := r.GetPublishedAt()

		logrus.Debugf("[%d] version: '%s' tag: '%s' date: %v",
			wID, newVersion, newTag, newDate)

		if prevState.PublishDate != nil && prevState.PublishDate.Unix() > newDate.Unix() {
			break // Old release
		}
		if newVersion != "" && prevState.Version == newVersion {
			break // Already seen
		}
		if (prevState.Tag != nil && *prevState.Tag == newTag) && prevState.Version == newVersion {
			break // Already seen
		}

		newReleaseList = append(newReleaseList, &Release{
			RepoState: &RepoState{
				Repo:        prevState.Repo,
				Version:     newVersion,
				Tag:         r.TagName,
				PreRelease:  r.Prerelease,
				PublishDate: r.PublishedAt,
				body:        r.Body,
			},
			Body: r.Body,
		})

		if prevState.PublishDate == nil {
			// It must be the first time this project is checked,
			// let's not list all releases.
			break
		}
	}

	return newReleaseList, nil
}
