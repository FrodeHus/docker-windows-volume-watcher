package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var watcher *fsnotify.Watcher
var container string
var rootPath string

func init() {
	flag.StringVar(&container, "container", "", "Name of the container instance that you wish to notify of filesystem changes")
	flag.StringVar(&rootPath, "path", "", "Root path where to watch for changes")
}

func main() {
	flag.Parse()
	watcher, _ = fsnotify.NewWatcher()

	defer watcher.Close()
	if rootPath == "" {
		rootPath = "."
	}

	if err := filepath.Walk(rootPath, watchDir); err != nil {
		fmt.Println("ERROR", err)
	}

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op == fsnotify.Write {
					notifyDocker(event)
				}
			case err := <-watcher.Errors:
				fmt.Println("Error: ", err)
			}
		}
	}()

	<-done
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
	fmt.Println("Updating container file ", containerPath)

	result, err := exec.Command("docker", "exec", container, "stat", "-c", "%a", containerPath).Output()
	if err != nil {
		fmt.Println("Error retrieving file permissions: ", err)
	}

	perms, err := strconv.Atoi(strings.TrimSuffix(string(result), "\n"))
	if err != nil {
		fmt.Println("Raw permissions: ", result)
		fmt.Println("Failed to convert permissions: ", err)
		return
	}

	_, err = exec.Command("docker", "exec", container, "/bin/sh", "-c", fmt.Sprintf("chmod %d %s", perms, containerPath)).Output()
	if err != nil {
		fmt.Printf("Error notifying container about file change: %v", err)
	}
}

func watchDir(path string, fi os.FileInfo, err error) error {
	if fi.Mode().IsDir() && !(strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules")) {
		fmt.Println("Watching ", path)
		return watcher.Add(path)
	}
	return nil
}
