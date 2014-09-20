package main

import (
	"fmt"
	"strings"
	"time"
	"os/exec"
	"os"
	"path/filepath"
)

var recordings = make(map[Showing]bool)

func Recorder(recordC <-chan Showing) (err error) {
	for {
		select {
		case showing := <-recordC:
			if _, ok := recordings[showing]; ok {
				fmt.Println("Already recording, skipping")
				continue
			}

			go record(showing)
		}
	}

	return
}

func record(showing Showing) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error: %s", r)
		}
	}()

	fmt.Printf("Recording %s (started at %s) is now recording (%s), ends at %s\n", showing.Title, showing.Start.Local(), timeNow().Local(), showing.End.Local())

	recordings[showing] = true
	defer func() {
		recordings[showing] = false
	}()

	args := getSlingArgs()

	filename := genFilename(showing, 0)

//	secsLeft := showing.End.Sub(timeNow()) / time.Second
//	if secsLeft < 0 {
//		secsLeft = 1
//	}
	args = append(args, "-output", filename)

	bin := fmt.Sprintf("%s/rec350.pl", filepath.Dir(os.Args[0]))
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		bin, err = filepath.Abs("rec350.pl")
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("  Running: ", bin, strings.Join(args, " "))

	cmd := exec.Command(bin, args...)

	// NOTE: $dur param to perl script was not working properly (quick glance shows they are counting based on packets, which Sling adapter may screw up)
	go func() {
		time.Sleep(showing.End.Sub(timeNow()))
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("Could not kill process: ", err.Error())
		}
	}()

	if err := cmd.Run(); err != nil && !strings.Contains(err.Error(), "signal: killed") {
		fmt.Println("Error running: ", err)
	}

	if showing.End.Sub(timeNow()) > 30 * time.Second {
		fmt.Println("Exited early, trying again")
		record(showing)
	}
}

func getSlingArgs() (args []string) {
	for k, v := range rawConfig {
		if !strings.HasPrefix(k, "sling") {
			continue
		}

		args = append(args, "-" + strings.ToLower(k[5:6]) + k[6:], fmt.Sprintf("%s", v))
	}

	return
}

func genFilename(showing Showing, ver int) string {
	filename := fmt.Sprintf("%s/%s - %s", config.RecordingDir, showing.Title, showing.Start.Local().Format("2006-01-02 3:04PM"))
	if showing.Title != showing.Subtitle {
		filename += ": " + showing.Subtitle
	}

	if ver > 100 {
		panic(fmt.Sprintf("Too many attempts to record showing: %#v", showing))
	}

	if ver > 0 {
		filename += fmt.Sprintf(" (part %d)", ver + 1)
	}

	filename += ".asf"

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return filename
	} else {
		return genFilename(showing, ver + 1)
	}
}