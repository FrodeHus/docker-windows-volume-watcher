package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var watcher *fsnotify.Watcher

var (
	container string
	rootPath  string
	delay     int

	ignoreArg string
	ignores   []string
)

func init() {
	flag.StringVar(&container, "container", "", "Name of the container instance that you wish to notify of filesystem changes")
	flag.StringVar(&rootPath, "path", "", "Root path where to watch for changes")
	flag.IntVar(&delay, "delay", 100, "Delay in milliseconds before notifying about a file that's changed")
	flag.StringVar(&ignoreArg, "ignore", "node_modules;vendor", "Semicolon-separated list of directories to ignore. "+
		"Glob expressions are supported.")
}

func main() {
	flag.Parse()

	ignores = strings.Split(ignoreArg, ";")

	watcher, _ = fsnotify.NewWatcher()
	defer watcher.Close()
	if rootPath == "" {
		rootPath = "."
	}

	if err := filepath.Walk(rootPath, watchDir); err != nil {
		fmt.Println("ERROR", err)
	}

	// Map of filenames we're currently notifying about.
	var processes sync.Map

	for {
		select {
		case event := <-watcher.Events:
			switch event.Op {
			case fsnotify.Write:
				if _, ok := processes.Load(event.Name); ok {
					continue
				}
				processes.Store(event.Name, nil)

				go func(event fsnotify.Event) {
					defer processes.Delete(event.Name)

					// Wait for further events to accomodate the way editors
					// save files.
					time.Sleep(time.Duration(delay) * time.Millisecond)

					// Ensure the file hasn't been renamed or removed.
					if _, ok := processes.Load(event.Name); !ok {
						return
					}

					notifyDocker(event)
				}(event)

			case fsnotify.Rename, fsnotify.Remove:
				processes.Delete(event.Name)
			}
		case err := <-watcher.Errors:
			fmt.Println("Error: ", err)
		}
	}
}

func notifyDocker(event fsnotify.Event) {
	if event.Op != fsnotify.Write {
		return
	}
	file := filepath.ToSlash(event.Name)

	containerPath := strings.TrimPrefix(file, rootPath)
	if strings.HasPrefix(containerPath, "/") {
		containerPath = strings.TrimPrefix(containerPath, "/")
	}
	fmt.Println("Updating container file", containerPath)

	_, err := exec.Command("docker", "exec", container, "/bin/sh", "-c", fmt.Sprintf("chmod $(stat -c %%a %s) %s", containerPath, containerPath)).Output()
	if err != nil {
		fmt.Printf("Error notifying container about file change: %v\n", err)
	}
}

func watchDir(path string, fi os.FileInfo, err error) error {
	if !fi.Mode().IsDir() {
		return nil
	}

	// Ignore hidden directories.
	if len(path) > 1 && strings.HasPrefix(path, ".") {
		return filepath.SkipDir
	}

	for _, pattern := range ignores {
		ok, err := filepath.Match(pattern, fi.Name())
		if err != nil {
			return err
		}
		if ok {
			return filepath.SkipDir
		}
	}

	fmt.Println("Watching ", path)
	return watcher.Add(path)
}
