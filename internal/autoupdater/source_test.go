package autoupdater

import (
	"context"
	"reflect"
	"testing"
)

func TestGitHubSourceAllPages(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote test")
	}
	const source = "https://github.com/inavFlight/inav-configurator"
	src, err := NewSource(source)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var token string
	for {
		releases, nextToken, err := src.AvailableVersions(ctx, token)
		if err != nil {
			t.Fatal(err)
		}
		if len(releases) == 0 {
			t.Fatal("no releases found before returning an empty token")
		}
		if nextToken == "" {
			break
		}
		token = nextToken
	}
}

func TestGitHubSources(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote test")
	}
	sources := []string{
		"https://github.com/inavFlight/inav",
		"https://github.com/inavFlight/inav-configurator",
		"https://github.com/opentx/opentx",
		"https://github.com/betaflight/betaflight-configurator",
	}
	for _, s := range sources {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			src, err := NewSource(s)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			releases, nextToken, err := src.AvailableVersions(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(releases) == 0 {
				t.Fatal("no releases found")
			}
			if nextToken == "" {
				t.Fatal("no nextToken returned")
			}
			releases2, nextToken2, err := src.AvailableVersions(ctx, nextToken)
			if err != nil {
				t.Fatal(err)
			}
			if len(releases2) == 0 {
				t.Fatal("no releases found on next page")
			}
			if nextToken2 == "" && len(releases2) == len(releases) {
				t.Fatal("no nextToken returned on next page")
			}
			if reflect.DeepEqual(releases, releases2) {
				t.Fatal("same releases returned with different token")
			}
		})
	}
}
