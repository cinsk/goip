package main

import (
	"os"
	"testing"
)

func TestDownloader_InvalidURL(env *testing.T) {
	outfile := func() string {
		d := Downloader{}
		defer d.Close()
		err := d.Fetch("http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip.blahblahblahblah")

		if err == nil {
			env.Errorf("error expected, but not found")
		}
		return d.Archive
	}()

	_, err := os.Stat(outfile)
	if os.IsExist(err) {
		os.Remove(outfile)
		env.Errorf("downloaded file exists, where it shouldn't: %v", outfile)
	}

}

func TestDownloader_URL(env *testing.T) {
	outfile := func() string {
		d := Downloader{}
		defer d.Close()
		err := d.Fetch("http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip")
		if err != nil {
			env.Errorf("error: %v", err)
		}

		err = d.Unpack()
		if err != nil {
			env.Errorf("error: %v", err)
		}

	}()

	_, err := os.Stat(outfile)
	if os.IsExist(err) {
		os.Remove(outfile)
		env.Errorf("downloaded file exists, where it shouldn't: %v", outfile)
	}

}
