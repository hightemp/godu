package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type DirInfo struct {
	Path string
	Size int64
}

var (
	humanReadableFlag bool
	maxDepth          int
	exclude           string
	excludePatterns   []string
	scanPath          string
	resultChan        chan DirInfo
)

func main() {

	flag.BoolVar(&humanReadableFlag, "h", false, "Output in human-readable format")
	flag.IntVar(&maxDepth, "d", -1, "Maximum depth of results output (-1 = no limit)")
	flag.StringVar(&exclude, "e", "", "Exclude directories matching pattern (comma-separated)")
	flag.Parse()

	if exclude != "" {
		excludePatterns = strings.Split(exclude, ",")
	}

	args := flag.Args()
	scanPath = "."
	if len(args) > 0 {
		scanPath = args[0]
	}

	info, err := os.Stat(scanPath)
	if err != nil {
		fmt.Println("Path doesn't exist")
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Println("Path doesn't exist")
		os.Exit(1)
	}

	resultChan = make(chan DirInfo, 1000)

	var resultWg sync.WaitGroup
	resultWg.Add(1)

	var dirResults []DirInfo
	go func() {
		defer resultWg.Done()
		for result := range resultChan {
			dirResults = append(dirResults, result)
		}
	}()

	size := scanDir(scanPath, 0)

	resultChan <- DirInfo{Path: scanPath, Size: size}

	close(resultChan)
	resultWg.Wait()

	sort.Slice(dirResults, func(i, j int) bool {
		return dirResults[i].Size > dirResults[j].Size
	})

	for _, dirResult := range dirResults {
		if humanReadableFlag {
			fmt.Printf("%s\t%s\n", formatSize(dirResult.Size), dirResult.Path)
		} else {
			fmt.Printf("%d\t%s\n", dirResult.Size, dirResult.Path)
		}
	}
}

func scanDir(path string, depthLevel int) int64 {
	var size int64
	var syncSize atomic.Int64
	var wg sync.WaitGroup

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Can't read dir: %s\n", path)
		return 0
	}

EntriesLoop:
	for _, entry := range entries {
		innerPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			for _, pattern := range excludePatterns {
				if matched, _ := filepath.Match(pattern, entry.Name()); matched {
					continue EntriesLoop
				}
			}

			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				subSize := scanDir(path, depthLevel+1)

				if depthLevel < maxDepth || maxDepth == -1 {
					resultChan <- DirInfo{Path: path, Size: subSize}
				}

				syncSize.Add(subSize)
			}(innerPath)

		} else {
			info, err := entry.Info()
			if err != nil {
				fmt.Printf("Can't get file info: %s\n", innerPath)
				continue
			}

			fileSize := info.Size()
			size += fileSize
		}
	}

	wg.Wait()

	size += syncSize.Load()

	return size
}

func formatSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2fT", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2fG", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2fM", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fK", float64(size)/KB)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
