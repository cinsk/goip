package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

type Downloader struct {
	URL     string
	Archive string
	Base    string
}

func (d *Downloader) Close() {
	if d.Base != "" {
		log.Printf("removing %v", d.Base)
		os.RemoveAll(d.Base)
	}
	if d.Archive != "" {
		log.Printf("removing %v", d.Archive)
		os.Remove(d.Archive)
	}
}

func (d *Downloader) Unpack() error {
	dir, err := ioutil.TempDir("/tmp", "geoip")
	if err != nil {
		return err
	}
	d.Base = dir

	r, err := zip.OpenReader(d.Archive)
	defer r.Close()
	if err != nil {
		return err
	}
	for _, f := range r.File {
		basename := path.Base(f.Name)
		log.Printf("decompressing: %v\n", basename)
		outname := path.Join(d.Base, path.Base(f.Name))

		src, err := f.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(outname)
		if err != nil {
			return err
		}
		defer dst.Close()
		io.Copy(dst, src)
	}
	return nil
}

func (d *Downloader) Fetch(url string) error {
	log.Printf("fetching url: %v", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("status: %v", resp.Status)
	if resp.StatusCode != 200 {
		return fmt.Errorf("non 200 status: %v", resp.Status)
	}

	out, err := ioutil.TempFile("/tmp", "geoarchive")
	if err != nil {
		return err
	}
	defer out.Close()
	log.Printf("saving to %v", out.Name())
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(out.Name())
		return err
	}

	d.URL = url
	d.Archive = out.Name()
	return nil
}

func smain() {
	// zipName, err := Download("http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip")
	d := Downloader{}
	defer d.Close()
	err := d.Fetch("http://geolite.maxmind.com/download/geoip/database/GeoLite2-City-CSV.zip")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	err = d.Unpack()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
