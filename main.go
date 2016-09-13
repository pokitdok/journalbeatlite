package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/pokitdok/libbeatlite"
)

const (
	DEFAULTCONFIG = "./config.json"
	DEFAULTCURSOR = "./cursor"
)

type jbconf struct {
	libbeatlite.Client        // embedding Client as anonymous field allows for the simple parsing of a config file
	ParseJson          bool   `json:"parse_json_messages"` // if True, try to extract json from log messages
	CursorFile         string `json:"cursor_file_name"`    // store last processed cursor in this file
	cursor             string
}

func configure(filename string) (*jbconf, error) {

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	config := new(jbconf)
	if err = json.Unmarshal(b, config); err != nil {
		return nil, fmt.Errorf("error parsing journalbeatlite config file: %v", err)
	}
	if config.CursorFile == "" {
		config.CursorFile = DEFAULTCURSOR
	}
	cursor, err := ioutil.ReadFile(config.CursorFile)
	if err == nil {
		config.cursor = strings.TrimSpace(string(cursor))
	}

	if config.Name == "" {
		config.Name = "journalbeatlite"
	}

	return config, nil
}

func tail(cursor string) (chan *sdjournal.JournalEntry, error) {

	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, fmt.Errorf("error opening journal: %v", err)
	}

	if cursor == "" {
		err = j.SeekHead()
	} else {
		err = j.SeekCursor(cursor)
	}
	if err != nil {
		return nil, fmt.Errorf("error seeking cursor: %v", err)
	}

	eventChan := make(chan *sdjournal.JournalEntry)

	go func() {
		defer j.Close()
		defer close(eventChan)
		for {
			c, err := j.Next()
			if err != nil {
				log.Fatal(err)
			}
			if c == 0 {
				j.Wait(sdjournal.IndefiniteWait)
				continue
			}
			e, err := j.GetEntry()
			if err != nil {
				log.Fatal(err)
			}
			eventChan <- e
		}
	}()

	return eventChan, nil
}

func format(c *jbconf, e *sdjournal.JournalEntry) *libbeatlite.Message {

	journal := make(map[string]interface{})
	journal["__CURSOR"] = e.Cursor

	// copy fields, where possible converting from string to int
	for k, v := range e.Fields {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			journal[k] = i
		} else {
			journal[k] = v
		}
	}

	message := journal["MESSAGE"].(string)
	delete(journal, "MESSAGE")

	source := map[string]interface{}{
		"@timestamp": time.Unix(0, int64(e.RealtimeTimestamp)*1000).Format("2006-01-02T15:04:05.000Z"),
		"journal":    journal,
		"message":    message,
	}

	if c.ParseJson {
		var s map[string]interface{}
		if err := json.Unmarshal([]byte(message), &s); err == nil {
			source["structured"] = s
		}
	}

	// use sha256 of the cursor as the message id
	h := sha256.Sum256([]byte(e.Cursor))
	i := fmt.Sprintf("%x", h)

	return &libbeatlite.Message{Source: source, Id: i}
}

func commit(filename, cursor string) error {
	// atomic write to the cursor file

	f, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("error resolving absolute path for cursor file: %v", err)
	}

	t, err := ioutil.TempFile(filepath.Dir(f), "cursor-")
	if err != nil {
		return fmt.Errorf("error creating cursor temp file: %v", err)
	}
	defer os.Remove(t.Name())

	_, err = t.Write([]byte(cursor + "\n"))
	if err != nil {
		return fmt.Errorf("error writing cursor to temp file: %v", err)
	}
	t.Close()

	// os.Rename is not atomic, syscall.Rename is on Linux. however, if source and
	// dest are on different volumes, syscall.Rename will fail; this is why I am
	// not writing the temp file to /tmp but to the same directory where the
	// coursor file lives.
	// https://groups.google.com/forum/#!topic/golang-nuts/ZjRWB8bMhv4
	err = syscall.Rename(t.Name(), f)
	if err != nil {
		return fmt.Errorf("error renaming temp cursor file %q to %q: %v", t.Name(), f, err)
	}

	return nil
}

const Version = "0.3.0"

var (
	LibBuildHash string
	BuildHash    string
	BuildDate    string
)

func main() {

	path := flag.String("config", "./config.json", "path to the config file; prints sample config file when config=''")
	version := flag.Bool("version", false, "print version information and exit")
	noop := flag.Bool("noop", false, "do not send data to elasticsearch or advance the cursor; implies -debug")
    debug := flag.Bool("debug", false, "turn on debugging output")
	flag.Parse()

	if *version {
		fmt.Printf("journalbeatlite\tversion: %q build: %q date: %q\n", Version, BuildHash, BuildDate)
		fmt.Printf("libbeatlite\tversion: %q build: %q\n", libbeatlite.Version, LibBuildHash)
        os.Exit(0)
	}

    if *path == "" {
        // print sample config file
		c := jbconf{CursorFile: DEFAULTCURSOR, Client: libbeatlite.Client{URL: "http://127.0.0.1:9200", Name: "journalbeatlite"}}
		b, _ := json.MarshalIndent(c, "", "    ")
		fmt.Println(string(b))
        os.Exit(0)
    }

	config, err := configure(*path)
	if err != nil {
        log.Fatal(err)
	}
	b, _ := json.Marshal(config)
	log.Println(string(b))

    if *debug {
        config.Debug=true
    }

	if *noop {
		config.Noop = true // Noop is embedded field from config.Client
		config.Debug = true
	}

	eventChan, err := tail(config.cursor)
	if err != nil {
		log.Fatalf("error connecting to the journal: %v", err)
	}

	for event := range eventChan {
		m := format(config, event)
		// because Client is an embedded anonymous field of jbconf, not
		// only its fields, but its methods can be accessed directly.
		// So the call below could be config.Send(m); i am leaving the
		// long form config.Client.Send(m) to make it clean what gets
		// called, but that is really neat!
		_, err := config.Client.Send(m)
		if err != nil {
			log.Fatal(err)
		}
		if !config.Noop {
			err = commit(config.CursorFile, event.Cursor)
			if err != nil {
				log.Fatalf("error updating cursor offset: %v", err)
			}
		}
	}

}
