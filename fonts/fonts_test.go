package fonts

import (
	"testing"

	"github.com/fiam/max7456tool/mcm"
)

func testFontOrigin(t *testing.T, origin FontOrigin) {
	fonts, err := origin.Fonts()
	if err != nil {
		t.Error(err)
	} else {
		if len(fonts) == 0 {
			t.Error("no fonts returned")
		}
		for _, f := range fonts {
			r, err := f.Open()
			if err != nil {
				t.Errorf("error opening font %s: %v", f.Name, err)
				continue
			}
			defer r.Close()
			dec, err := mcm.NewDecoder(r)
			if err != nil {
				t.Errorf("error decoding font %s: %v", f.Name, err)
				continue
			}
			if dec.NChars() == 0 {
				t.Errorf("font %s has no chars", f.Name)
				continue
			}
			t.Logf("font %s from %s has %d chars", f.Name, f.URL, dec.NChars())
		}
	}
}

func testGHFontOrigin(t *testing.T, name string, url string) {
	origin := &gitHubDirFontOrigin{name: name, dirURL: url}
	if origin.Name() != name {
		t.Errorf("unexpected name %q, expecting %q", origin.Name(), name)
	}
	testFontOrigin(t, origin)
}

func TestINAVFonts(t *testing.T) {
	testGHFontOrigin(t, "INAV", "https://github.com/iNavFlight/inav-configurator/resources/osd")
}

func TestBetaflightFonts(t *testing.T) {
	testGHFontOrigin(t, "Betaflight", "https://github.com/betaflight/betaflight-configurator/resources/osd/2")
}

func TestOrigins(t *testing.T) {
	for _, origin := range Origins() {
		testFontOrigin(t, origin)
	}
}
