package fonts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/google/go-github/github"
)

type Font struct {
	Name string
	URL  string
}

func (f Font) Open() (io.ReadCloser, error) {
	resp, err := http.Get(f.URL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("invalid HTTP response code %d", resp.StatusCode)
	}
	return resp.Body, nil
}

type FontOrigin interface {
	Name() string
	Fonts() ([]Font, error)
}

var _ FontOrigin = (*gitHubDirFontOrigin)(nil)

type gitHubDirFontOrigin struct {
	name   string
	dirURL string
}

func (o *gitHubDirFontOrigin) Name() string {
	return o.name
}

func (o *gitHubDirFontOrigin) Fonts() ([]Font, error) {
	u, err := url.Parse(o.dirURL)
	if err != nil {
		return nil, err
	}
	if u.Hostname() != "github.com" {
		return nil, fmt.Errorf("host is %q instead of github.com", u.Hostname())
	}
	parts := strings.Split(u.Path[1:], "/")
	repoPath := strings.Join(parts[2:], "/")
	c := github.NewClient(nil)
	_, dirContents, _, err := c.Repositories.GetContents(context.Background(), parts[0], parts[1], repoPath, nil)
	if err != nil {
		return nil, err
	}
	var fonts []Font
	for _, entry := range dirContents {
		filename := entry.GetName()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".mcm" {
			continue
		}
		nonExt := filename[:len(filename)-len(ext)]
		capitalize := true
		var name []rune
		for _, c := range nonExt {
			if c == '_' || c == '-' || c == ' ' {
				capitalize = true
				if len(name) == 0 || name[len(name)-1] != ' ' {
					name = append(name, ' ')
				}
				continue
			}
			if capitalize {
				c = unicode.ToUpper(c)
				capitalize = false
			}
			name = append(name, c)
		}
		fonts = append(fonts, Font{
			Name: string(name),
			URL:  entry.GetDownloadURL(),
		})
	}
	sort.Slice(fonts, func(i, j int) bool {
		if fonts[i].Name == "Default" {
			return true
		}
		if fonts[j].Name == "Default" {
			return false
		}
		return fonts[i].Name < fonts[j].Name
	})
	return fonts, nil
}

func Origins() []FontOrigin {
	return []FontOrigin{
		&gitHubDirFontOrigin{"INAV", "https://github.com/iNavFlight/inav-configurator/resources/osd"},
		// v2 for BF are for BF >= 4.1
		&gitHubDirFontOrigin{"Betaflight", "https://github.com/betaflight/betaflight-configurator/resources/osd/2"},
	}
}
