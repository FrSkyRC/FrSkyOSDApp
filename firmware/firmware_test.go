package firmware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFirmware(t *testing.T) {
	f1 := &Firmware{
		URL: "https://github.com/FrSkyRC/PixelOSD/blob/master/firmware/FrSkyOSD-v1.0.0_20191025.bin",
	}

	name1, err := f1.VersionName()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, name1, "1.0.0")

	date1, err := f1.Date()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, date1, time.Date(2019, 10, 25, 0, 0, 0, 0, time.UTC))

	f2 := &Firmware{
		URL: "https://github.com/FrSkyRC/PixelOSD/blob/master/firmware/FrSkyOSD-v1.99.0_20200615.bin",
	}

	name2, err := f2.VersionName()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, name2, "2.0.0-beta.1")

	date2, err := f2.Date()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, date2, time.Date(2020, 06, 15, 0, 0, 0, 0, time.UTC))
}
