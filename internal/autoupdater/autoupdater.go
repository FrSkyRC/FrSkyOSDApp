package autoupdater

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pkg/browser"
)

type Options struct {
	// Version overrides the app version returned automatically
	Version         string
	AcceptPreleases bool
	// NoSkipRelease disables the "Skip this release" option
	NoSkipRelease bool
	Source        Source
	Dialog        Dialog
}

type AutoUpdater struct {
	opts *Options
}

// New returns a new AutoUpdater. See the Options
// type for the available options.
func New(opts *Options) (*AutoUpdater, error) {
	if opts == nil || opts.Source == nil {
		panic(errors.New("Source cannot be nil"))
	}
	if opts == nil || opts.Dialog == nil {
		panic(errors.New("Dialog cannot be nil"))
	}
	return &AutoUpdater{
		opts: opts,
	}, nil
}

func (au *AutoUpdater) currentVersion() (string, error) {
	return au.opts.Version, nil
}

func (au *AutoUpdater) CheckForUpdates(ctx context.Context) error {
	var allReleases []*Release
	var releases []*Release
	var next string
	var err error
	for {
		releases, next, err = au.opts.Source.AvailableVersions(ctx, next)
		if err != nil {
			return err
		}
		foundNonPrerelease := false
		for _, r := range releases {
			if !r.IsPrerelease {
				foundNonPrerelease = true
			}
			allReleases = append(allReleases, r)
		}
		if foundNonPrerelease {
			break
		}
		if next == "" {
			break
		}
	}
	if !au.opts.AcceptPreleases {
		var nonPrereleases []*Release
		for _, r := range allReleases {
			if !r.IsPrerelease {
				nonPrereleases = append(nonPrereleases, r)
			}
		}
		allReleases = nonPrereleases
	}
	if len(allReleases) == 0 {
		// No updates found
		return nil
	}
	sort.Slice(allReleases, func(i, j int) bool {
		ri := allReleases[i]
		rj := allReleases[j]
		vi := version.Must(version.NewVersion(ri.Version))
		vj := version.Must(version.NewVersion(rj.Version))

		return vi.GreaterThan(vj)
	})
	newestRelease := allReleases[0]
	currentVersion, err := au.currentVersion()
	if err != nil {
		return err
	}
	cv, err := version.NewVersion(currentVersion)
	if err != nil {
		return err
	}
	nv, err := version.NewVersion(newestRelease.Version)
	if err != nil {
		return err
	}
	if nv.LessThanOrEqual(cv) {
		// Latest update is the version we're already
		// running
		return nil
	}
	var responses []DialogResponse
	if au.opts.NoSkipRelease {
		responses = append(responses, DialogResponseCancel)
	} else {
		responses = append(responses, DialogResponseSkipRelease)
	}
	responses = append(responses, DialogResponseRemindLater)
	if au.canInstallUpdates() {
		responses = append(responses, DialogResponseDownloadAndInstall)
	} else {
		responses = append(responses, DialogResponseDownload)
	}
	var resp DialogResponse
	var wg sync.WaitGroup
	wg.Add(1)
	opts := &DialogOptions{
		CurrentVersion:   currentVersion,
		AvailableRelease: newestRelease,
		Responses:        responses,
		Response: func(r DialogResponse) {
			defer wg.Done()
			resp = r
		},
	}
	go au.opts.Dialog.ShowUpdaterDialog(opts)
	wg.Wait()
	switch resp {
	case DialogResponseCancel:
		return nil
	case DialogResponseSkipRelease:
		return au.skipRelease(opts.AvailableRelease)
	case DialogResponseRemindLater:
		// Nothing to do, we'll remind on next check
		return nil
	case DialogResponseDownload:
		return browser.OpenURL(opts.AvailableRelease.URL)
	case DialogResponseDownloadAndInstall:
		// TODO
	}
	return nil
}

func (au *AutoUpdater) skipRelease(rel *Release) error {
	// TODO: Skip
	return nil
}

// ScheduleCheckingForUpdates checks for updates once, then starts
// checking again indefinitely at the given interval.
func (au *AutoUpdater) ScheduleCheckingForUpdates(interval time.Duration) {
	doCheck := func() {
		ctx := context.Background()
		if err := au.CheckForUpdates(ctx); err != nil {
			log.Printf("error checking for updates: %v", err)
		}
	}
	go doCheck()
	for range time.Tick(interval) {
		doCheck()
	}
}
