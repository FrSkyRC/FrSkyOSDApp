package firmware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"

	"osdapp/internal/osdversion"
)

const (
	firmwareExtension            = ".bin"
	firmwareNotesExtension       = ".md"
	firmwarePrefix               = "FrSkyOSD-v"
	firmwareVersionDateSeparator = "_"
	firmwareSourceURL            = "https://github.com/FrSkyRC/PixelOSD/firmware"
)

var (
	errMissingSeparator = errors.New("missing version-date separator")
)

// Firmware represents an available firmware with its
// download URL and the URL for its release notes
type Firmware struct {
	URL             string
	ReleaseNotesURL string
}

// Filename returns the filename of the firmware
func (f *Firmware) Filename() (string, error) {
	u, err := url.Parse(f.URL)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}

func (f *Firmware) basename() (string, error) {
	name, err := f.Filename()
	if err != nil {
		return "", err
	}
	if filepath.Ext(name) != firmwareExtension {
		return "", fmt.Errorf("firmware file %q has incorrect extension %q instead of %q",
			name, filepath.Ext(name), firmwareExtension)
	}
	if !strings.HasPrefix(name, firmwarePrefix) {
		return "", fmt.Errorf("filename %q doesn't look like a valid firmware", name)
	}
	return name, nil
}

func (f *Firmware) version() (major, minor, patch int, err error) {
	name, err := f.basename()
	if err != nil {
		return -1, -1, -1, err
	}
	nonPrefix := name[len(firmwarePrefix):]
	sep := strings.Index(nonPrefix, firmwareVersionDateSeparator)
	if sep < 0 {
		return -1, -1, -1, errMissingSeparator
	}
	version := nonPrefix[:sep]
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return -1, -1, -1, err
	}
	return major, minor, patch, nil
}

// VersionName returns the version name that should
// be display to the user
func (f *Firmware) VersionName() (string, error) {
	major, minor, patch, err := f.version()
	if err != nil {
		return "", err
	}
	return osdversion.Format(major, minor, patch), nil
}

// Date returns the release date as parsed from the filename
func (f *Firmware) Date() (time.Time, error) {
	name, err := f.basename()
	if err != nil {
		return time.Time{}, err
	}
	ext := filepath.Ext(name)
	nonExt := name[:len(name)-len(ext)]
	nonPrefix := nonExt[len(firmwarePrefix):]
	sep := strings.Index(nonPrefix, firmwareVersionDateSeparator)
	if sep < 0 {
		return time.Time{}, errMissingSeparator
	}
	date := nonPrefix[sep+1:]
	return time.Parse("20060102", date)
}

// Load checks the available firmwares in GitHub and
// returns their URLs
func Load() ([]*Firmware, error) {
	u, err := url.Parse(firmwareSourceURL)
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
	candidates := make(map[string]string)
	var firmwares []*Firmware
	for _, entry := range dirContents {
		filename := entry.GetName()
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case firmwareExtension:
			candidates[filename] = entry.GetDownloadURL()
		case firmwareNotesExtension:
			candidates[filename] = entry.GetHTMLURL()
		}
	}
	for k, v := range candidates {
		ext := strings.ToLower(filepath.Ext(k))
		if ext != firmwareExtension {
			continue
		}
		nonExt := k[:len(k)-len(ext)]
		notes := candidates[nonExt+firmwareNotesExtension]
		if notes == "" {
			log.Printf("ignoring candidate %q, missing release notes", k)
			continue
		}
		f := &Firmware{
			URL:             v,
			ReleaseNotesURL: notes,
		}
		if _, err := f.VersionName(); err != nil {
			log.Printf("ignoring candidate %q, can't find version name: %v", k, err)
			continue
		}
		if _, err := f.Date(); err != nil {
			log.Printf("ignoring candidate %q, can't find version date: %v", k, err)
			continue
		}
		firmwares = append(firmwares, f)
	}
	sort.Slice(firmwares, func(i, j int) bool {
		d1, err := firmwares[i].Date()
		if err != nil {
			panic(err)
		}
		d2, err := firmwares[j].Date()
		if err != nil {
			panic(err)
		}
		// Reverse chronological order
		return d2.Before(d1)
	})
	return firmwares, nil
}
