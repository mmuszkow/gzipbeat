package main

import (
    "bufio"
	"fmt"
    "compress/gzip"
    "io/ioutil"
	"time"
    "os"
    "path/filepath"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
)

// Gzipbeat implements Beater interface and sends Nginx status using libbeat.
type Gzipbeat struct {
	// Holds configuration of gzipbeat parsed by libbeat.
	config ConfigSettings
	// Beat client.
	events publisher.Client
    // Local machine hostname.
    hostname string
    // List of files that were sent successfully.
    registry []string
}

// Config Gzipbeat according to gzipbeat.yml.
func (gb *Gzipbeat) Config(b *beat.Beat) error {
	err := cfgfile.Read(&gb.config, "")
	if err != nil {
		logp.Err("Error reading configuration file: %v", err)
		return err
	}

	return nil
}

// Setup Gzipbeat.
func (gb *Gzipbeat) Setup(b *beat.Beat) error {
	gb.events = b.Events

    // read registry if file exists
    _, err := os.Stat(gb.config.Registry)
    if err == nil || os.IsExist(err) {
        // read content
        rr, err := os.Open(gb.config.Registry)
        if err != nil {
            logp.Err("Error opening registry file: %v", err)
            return err
        }
        defer rr.Close()

        scanner := bufio.NewScanner(rr)    
        for scanner.Scan() {
            gb.registry = append(gb.registry, scanner.Text())
        }
        err = scanner.Err()
        if err != nil {
            logp.Err("Error scanning registry file: %v", err)
            return err
        }
    }

    // get hostname to be later used in events
    hostname, err := os.Hostname()
    if err != nil {
        logp.Err("Error getting hostname: %v", err)
        return err
    }
    gb.hostname = hostname

	return nil
}

// Cleanup Gzipbeat.
func (gb *Gzipbeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop Gzipbeat.
func (gb *Gzipbeat) Stop() {
}

// Sends a single logfile, returns false if we should retry
func send(gb *Gzipbeat, filename string, fields *map[string]string) {
    // check if we have access to file
    fi, err := os.Open(filename)
    if err != nil {
        logp.Err("Error opening file %s: %v", filename, err)
        return
    }
    defer fi.Close()

    // gunzip content
    fz, err := gzip.NewReader(fi)
    if err != nil {
        logp.Err("Error gunzipping file %s: %v", filename, err)
        return
    }
    defer fz.Close()

    content, err := ioutil.ReadAll(fz)
    if err != nil {
        logp.Err("Error reading file %s: %v", filename, err)
        return
    }

    // publish new event
    event := common.MapStr{
        "@timestamp": common.Time(time.Now()),
		"type":       "gzipfile",
		"file":       filename,
        "content":    string(content),
        "host":       gb.hostname,
    }
    if fields != nil {
        for name,val := range *fields {
            event[name] = val
        }
    }
    gb.events.PublishEvent(event, publisher.Sync)
}

// Saves to (mem & disk) registry that we processed single logfile
func saveToRegistry(gb *Gzipbeat, filename string) error {
    ra, err := os.OpenFile(gb.config.Registry, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
    if err != nil {
        logp.Err("Error opening registry file: %v", err)
        return err
    }
    defer ra.Close()

    _, err = fmt.Fprintln(ra, filename)
    return err
}

// Run Gzipbeat.
func (gb *Gzipbeat) Run(b *beat.Beat) error {

    // iterate through each config section
  	for _,input := range gb.config.Input {

        // list all gzip files in directory
        gzips, _ := filepath.Glob(input.Path)
        if input.Exclude != "" {
            exclude, _ := filepath.Glob(input.Exclude)
            gzips = diff(gzips, exclude)
        }
        gzips = diff(gzips, gb.registry)

        // do 1 file at the time
        for _,filename := range gzips {
            send(gb, filename, &input.Fields)
            err := saveToRegistry(gb, filename)
            if err != nil {
                logp.Err("Error saving to registry file %s: %v", gb.config.Registry, err)
                return err
            }
        }
 	}

	return nil
}

// Go doesn't have a diff function
func diff(slice1 []string, slice2 []string) []string {
    var diff []string

    // Loop two times, first to find slice1 strings not in slice2,
    // second loop to find slice2 strings not in slice1
    for i := 0; i < 2; i++ {
        for _, s1 := range slice1 {
            found := false
            for _, s2 := range slice2 {
                if s1 == s2 {
                    found = true
                    break
                }
            }
            // String not found. We add it to return slice
            if !found {
                diff = append(diff, s1)
            }
        }
        // Swap the slices, only if it was the first loop
        if i == 0 {
            slice1, slice2 = slice2, slice1
        }
    }

    return diff
}

