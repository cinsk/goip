package main

import (
	"os"
	"path"
	"testing"
)

func IsFileExist(filename string) bool {
	_, err := os.Stat(filename)
	return os.IsExist(err)
}

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

	if IsFileExist(outfile) {
		os.Remove(outfile)
		env.Errorf("downloaded file exists, where it shouldn't: %v", outfile)
	}

}

func TestDownloader_URL(env *testing.T) {
	dir, zip := func() (string, string) {
		d := Downloader{}
		defer d.Close()

		err := d.Fetch(GEOLITE_ARCHIVE_URL)
		if err != nil {
			env.Errorf("error: %v", err)
		}

		err = d.Unpack()
		if err != nil {
			env.Errorf("error: %v", err)
		}
		return d.Base, d.Archive
	}()

	if IsFileExist(dir) {
		os.Remove(dir)
		env.Errorf("Temp directory exists, where it shouldn't: %v", dir)
	}
	if IsFileExist(zip) {
		os.Remove(zip)
		env.Errorf("the downloaded archive exists, where it shouldn't: %v", zip)
	}
}

func TestDownloader_CheckDatabaseIntegrity(env *testing.T) {
	d := Downloader{}
	defer d.Close()

	err := d.Fetch(GEOLITE_ARCHIVE_URL)
	if err != nil {
		env.Errorf("error: %v", err)
	}

	err = d.Unpack()
	if err != nil {
		env.Errorf("error: %v", err)
	}

	citydb := path.Join(d.Base, GEOLITE_CITY_CSV_FILE)
	if !IsFileExist(citydb) {
		env.Errorf("City CSV file not found: %v", citydb)
	}
	blksdb := path.Join(d.Base, GEOLITE_BLOCK_CSV_FILE)
	if !IsFileExist(blksdb) {
		env.Errorf("City CSV file not found: %v", blksdb)
	}
}
