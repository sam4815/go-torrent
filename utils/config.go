package utils

import "flag"

var (
	initialized bool
	debug       bool
	filePath    string
)

func InitFlags() {
	flag.BoolVar(&debug, "debug", false, "enable debug logs")
	flag.StringVar(&filePath, "file", "", "torrent file path")

	flag.Parse()

	initialized = true
}

func GetDebug() bool {
	if !initialized {
		InitFlags()
	}

	return debug
}

func GetFilePath() string {
	if !initialized {
		InitFlags()
	}

	return filePath
}
