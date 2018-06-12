package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var watcher *fsnotify.Watcher

func main() {
	watcher, _ = fsnotify.NewWatcher()
	defer watcher.Close()

	if err := filepath.Walk(".", watchDir); err != nil {
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

	result, err := exec.Command("docker", "exec", "radix-web-dev_container", "stat", "-c", "%a", event.Name).Output()
	if strings.Contains(event.Name, ".git") {
		return
	}

	perms, err := strconv.Atoi(strings.TrimSuffix(string(result), "\n"))
	if err != nil {
		fmt.Println("Failed to convert permissions: ", err)
		return
	}
	_, err = exec.Command("docker", "exec", "radix-web-dev_container", "/bin/sh", "-c", fmt.Sprintf("chmod %d %s", perms, event.Name)).Output()
	if err != nil {
		fmt.Printf("Error notifying container about file change: %v", err)
	}
	fmt.Println(fmt.Sprintf("%s: %s", event.Op, event.Name))
}

func watchDir(path string, fi os.FileInfo, err error) error {
	if fi.Mode().IsDir() && !(strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules")) {
		fmt.Println("Watching ", path)
		return watcher.Add(path)
	}
	return nil
}
