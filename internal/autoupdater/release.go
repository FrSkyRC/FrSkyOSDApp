package autoupdater

// Asset represents a release asset that can be downloaded
type Asset struct {
	Name string
	URL  string
}

// Release represents a release of the app
type Release struct {
	Version      string
	IsPrerelease bool
	Notes        string
	NotesHTML    string
	URL          string
	Assets       []*Asset
}
