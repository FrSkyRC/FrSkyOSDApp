package autoupdater

import "fmt"

type DialogOptions struct {
	CurrentVersion   string
	AvailableRelease *Release
	Responses        []DialogResponse
	Response         func(DialogResponse)
}

func (opts *DialogOptions) AllowsResponse(r DialogResponse) bool {
	for _, v := range opts.Responses {
		if v == r {
			return true
		}
	}
	return false
}

type DialogResponse int

const (
	DialogResponseSkipRelease DialogResponse = iota
	DialogResponseCancel
	DialogResponseRemindLater
	DialogResponseDownload
	DialogResponseDownloadAndInstall
)

func (r DialogResponse) String() string {
	switch r {
	case DialogResponseSkipRelease:
		return "Skip this release"
	case DialogResponseCancel:
		return "Cancel"
	case DialogResponseRemindLater:
		return "Remind me later"
	case DialogResponseDownload:
		return "Download"
	case DialogResponseDownloadAndInstall:
		return "Download and install"
	}
	return fmt.Sprintf("unknown %T = %d", r, int(r))
}

type Dialog interface {
	ShowUpdaterDialog(opts *DialogOptions)
}
