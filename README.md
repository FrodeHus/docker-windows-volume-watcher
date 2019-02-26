# docker-windows-volume-watcher

Simple utility to get HMR working with WebPack inside containers running on Docker for Windows.

Inspired by [Mikhail Erofeev's Python tool](https://github.com/merofeev/docker-windows-volume-watcher)

## Usage

Run this tool in the root folder of your source.

`docker-windows-volume-watcher -container=[name of the container your volume is mounted in]`

### Arguments

```
Usage of docker-windows-volume-watcher:

  -container string
        Name of the container instance that you wish to notify of filesystem changes

  -delay int
        Delay in milliseconds before notifying about a file that's changed (default 100)

  -ignore string
        Semicolon-separated list of directories to ignore. Glob expressions are supported. (default "node_modules;vendor")

  -path string
        Root path where to watch for changes
```
