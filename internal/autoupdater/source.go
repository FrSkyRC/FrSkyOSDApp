package autoupdater

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/hashicorp/go-version"
)

// Source is an interface that provides the available releases
// for the given app. Use NewSource() to create a Source from
// the builtin ones or provide your own.
type Source interface {
	AvailableVersions(ctx context.Context, token string) (releases []*Release, nextToken string, err error)
}

// GitHubSource checks for releases from the given GitHub repository
type GitHubSource struct {
	Owner string
	Repo  string
}

func (s *GitHubSource) parseTag(tag string) (string, error) {
	if tag == "" {
		return "", errors.New("tag is empty")
	}
	strippedPrefixes := []string{
		"release/",
	}
	var vers string
	if tag[0] == 'v' || tag[0] == 'V' {
		vers = tag[1:]
	} else {
		lowerTag := strings.ToLower(tag)
		for _, p := range strippedPrefixes {
			if strings.HasPrefix(lowerTag, p) {
				vers = tag[len(p):]
				break
			}
		}
		if vers == "" {
			vers = tag
		}
	}
	v, err := version.NewVersion(vers)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

// AvailableVersions implements the Source interface
func (s *GitHubSource) AvailableVersions(ctx context.Context, token string) ([]*Release, string, error) {
	client := github.NewClient(nil)
	page := 0
	if token != "" {
		nextPage, err := strconv.Atoi(token)
		if err != nil {
			return nil, "", fmt.Errorf("invalid next token %q: %v", token, err)
		}
		page = nextPage
	}
	opts := &github.ListOptions{
		Page: page,
	}
	ghReleases, resp, err := client.Repositories.ListReleases(ctx, s.Owner, s.Repo, opts)
	if err != nil {
		return nil, "", err
	}
	releases := make([]*Release, 0, len(ghReleases))
	for _, r := range ghReleases {
		tag := r.GetTagName()
		vers, err := s.parseTag(tag)
		if err != nil {
			log.Printf("error parsing tag %q: %v, skipping", tag, err)
			continue
		}
		assets := make([]*Asset, len(r.Assets))
		for ii, a := range r.Assets {
			assets[ii] = &Asset{
				Name: a.GetName(),
				URL:  a.GetBrowserDownloadURL(),
			}
		}
		releases = append(releases, &Release{
			Version:      vers,
			IsPrerelease: r.GetPrerelease(),
			Notes:        r.GetBody(),
			URL:          r.GetHTMLURL(),
			Assets:       assets,
		})
	}
	nextToken := ""
	if resp.NextPage > 0 {
		nextToken = strconv.Itoa(resp.NextPage)
	}
	return releases, nextToken, nil
}

// NewSource finds a suitable source from the given origin
// and returns it.
func NewSource(origin string) (Source, error) {
	u, err := url.Parse(origin)
	if err == nil {
		if u.Hostname() == "github.com" && u.Path != "" {
			parts := strings.Split(u.Path[1:], "/")
			if len(parts) == 2 {
				return &GitHubSource{
					Owner: parts[0],
					Repo:  parts[1],
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("could not create a Source from %q", origin)
}
