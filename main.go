package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/DHowett/ranger"
	"github.com/dustin/go-humanize"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	sourceURL  string // download URL
	remoteFile string // remote file name
	localFile  string // local file name
	timeout    int    // timeout
	verbose    bool   // verbose mode shows a progress bar
	showFiles  bool   // list the files in the zip then exit
	limitBytes uint64 // limit the download to this many bytes
)

const defaultBufferSize = 128 * 1024

func init() {
	flag.StringVar(&sourceURL, "u", "", "the url you wish to download from")
	flag.StringVar(&remoteFile, "r", "", "the remote filename to download")
	flag.StringVar(&localFile, "o", "", "the output filename")
	flag.IntVar(&timeout, "t", 5, "timeout, in seconds")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&showFiles, "l", false, "list files in zip")
	flag.Uint64Var(&limitBytes, "b", 0, "limit filesize downloaded (in bytes)")

	flag.Parse()

	if sourceURL == "" {
		fmt.Println("You must specify a URL")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !showFiles {
		if remoteFile == "" {
			fmt.Println("You must specify a remote filename")
			flag.PrintDefaults()
			os.Exit(1)
		}

		if localFile == "" {
			_, localFile = filepath.Split(remoteFile)
		}
	}
}

// returns a progress bar fitting the terminal width given a progress percentage
func progressBar(progress int) (progressBar string) {

	var width int

	if runtime.GOOS == "windows" {
		// we'll just assume it's standard terminal width
		width = 80
	} else {
		width, _, _ = terminal.GetSize(0)
	}

	// take off 40 for extra info (e.g. percentage)
	width = width - 40

	// get the current progress
	currentProgress := (progress * width) / 100

	progressBar = "["

	// fill up progress
	for i := 0; i < currentProgress; i++ {
		progressBar = progressBar + "="
	}

	progressBar = progressBar + ">"

	// fill the rest with spaces
	for i := width; i > currentProgress; i-- {
		progressBar = progressBar + " "
	}

	// end the progressbar
	progressBar = progressBar + "] " + fmt.Sprintf("%3d%%", progress)

	return progressBar
}

func getBufferSize(lim uint64) uint64 {
	if lim < defaultBufferSize {
		return lim
	}

	return defaultBufferSize
}

func downloadFile(file *zip.File, writer *os.File) error {
	rc, err := file.Open()

	if err != nil {
		return err
	}

	defer rc.Close()

	downloaded := uint64(0)

	var filesize uint64
	var buf []byte

	if limitBytes != 0 {
		filesize = limitBytes
		buf = make([]byte, getBufferSize(limitBytes))
	} else {
		filesize = file.UncompressedSize64
		buf = make([]byte, defaultBufferSize)
	}

	humanizedFilesize := humanize.Bytes(filesize)

	for {
		// adjust the size of the buffer to get the exact
		// number of bytes we want to download
		if downloaded+defaultBufferSize > filesize {
			buf = make([]byte, filesize-downloaded)
		}

		if n, err := io.ReadFull(rc, buf); n > 0 && err == nil || err == io.EOF {
			writer.Write(buf[:n])
			downloaded += uint64(n)

			if verbose {
				fmt.Printf(
					"\r%s %10s/%-10s",
					progressBar(int(downloaded*100/filesize)),
					humanize.Bytes(downloaded),
					humanizedFilesize,
				)
			}

			if limitBytes != 0 && downloaded >= limitBytes {
				break
			}

		} else if err != nil {
			return err
		} else {
			break
		}
	}

	if verbose {
		fmt.Println()
	}

	return err
}

func findFile(reader *zip.Reader, filename string) (*zip.File, error) {
	if reader.File == nil {
		return nil, errors.New("file read error")
	}

	for _, f := range reader.File {
		if f.Name == filename {
			return f, nil
		}
	}

	return nil, errors.New("unable to find file")
}

func listFiles(reader *zip.Reader) error {
	if reader.File == nil {
		return errors.New("file read error")
	}

	var total uint64

	for _, f := range reader.File {
		total += f.UncompressedSize64
		fmt.Printf("%6s \t %s\n", humanize.Bytes(f.UncompressedSize64), f.Name)
	}

	fmt.Println("------")
	fmt.Printf("%6s\n", humanize.Bytes(total))

	return nil
}

func main() {
	downloadURL, err := url.Parse(sourceURL)

	reader, err := ranger.NewReader(
		&ranger.HTTPRanger{
			URL: downloadURL,
			Client: &http.Client{
				Timeout: time.Duration(timeout) * time.Second,
			},
		},
	)

	if err != nil {
		fmt.Printf("Unable to create reader for url: %s\n", downloadURL)
		os.Exit(1)
	}

	readerLen, err := reader.Length()

	if err != nil {
		fmt.Println("Unable to get reader length")
		os.Exit(1)
	}

	zipReader, err := zip.NewReader(reader, readerLen)
	if err != nil {
		fmt.Printf("Unable to create zip reader for url: %s\n", downloadURL)
		os.Exit(1)
	}

	if showFiles {
		listFiles(zipReader)
		return
	}

	var localFileHandle *os.File

	if localFile != "-" {
		localFileHandle, err = os.Create(localFile)
	} else {
		localFileHandle = os.Stdout
	}

	defer localFileHandle.Close()

	if err != nil {
		fmt.Printf("Unable to create local file: %s", localFile)
		os.Exit(1)
	}

	foundFile, err := findFile(zipReader, remoteFile)

	if err != nil {
		fmt.Printf("Unable find file: %s in zip.", remoteFile)
		os.Exit(1)
	}

	err = downloadFile(foundFile, localFileHandle)

	if err != nil {
		fmt.Printf("Unable read file %s from zip.", remoteFile)
		os.Exit(1)
	}
}
