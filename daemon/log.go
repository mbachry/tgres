//
// Copyright 2015 Gregory Trubetskoy. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var timeNow = func() time.Time {
	return time.Now()
}

var osRename = func(a, b string) error {
	return os.Rename(a, b)
}

var renameLogFile = func() {
	logDir, logFile := filepath.Split(Cfg.LogPath)
	filename := timeNow().Format(logFile + "-20060102_150405")
	fullpath := filepath.Join(logDir, filename)
	log.Printf("Starting new log file, current log archived as: '%s'", fullpath)
	osRename(Cfg.LogPath, fullpath)
}

var cycleLogFile = func() {

	if logFile != nil {
		renameLogFile()
	}

	file, err := os.OpenFile(Cfg.LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0666) // open with O_SYNC
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Unable to open log file '%s', %s\n", Cfg.LogPath, err)
		os.Exit(1)
	}

	log.SetOutput(file)
	if logFile != nil {
		logFile.Close()
	}
	logFile = file
}

var logFileCycler = func() {

	cycleLogFile() // Initial cycle

	go func() { // Wait for a cycle signal
		for {
			_ = <-cycleLogCh
			if quitting {
				// TODO This is not safe - "quitting" is not protected,
				// but we should not exit in the middle of a clean up...
				return
			}
			cycleLogFile()
		}
	}()

	go func() { // Periodic cycling
		for {
			time.Sleep(Cfg.LogCycle.Duration)
			cycleLogCh <- 1
			if quitting {
				return
			}
		}
	}()
}

// Write to both stderr and log
func logFatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	if Cfg.PidPath != "" && gracefulChildPid == 0 {
		os.Remove(Cfg.PidPath)
	}
	log.Fatalf(format, v...)
}
