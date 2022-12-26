/*
Copyright © 2022 HeavyDinosaur
*/
package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Files struct {
	FilePath     string
	Type         string
	Error        error
	UploadResult UploadResponse
}

type UploadFile struct {
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Source   string `json:"source"`
}

var (
	originalUrl bool
	largeUrl    bool
	mediumUrl   bool
	thumbUrl    bool
)

var apiKey string

const MaxNumOfWorkers = 1

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload images",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey = viper.GetString("api-key")
		uploadMain(args)
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().BoolVar(&originalUrl, "original", false, "Print original url")
	uploadCmd.Flags().BoolVar(&largeUrl, "large", false, "Print large url")
	uploadCmd.Flags().BoolVar(&mediumUrl, "medium", false, "Print medium url")
	uploadCmd.Flags().BoolVar(&thumbUrl, "thumb", false, "Print thumb url")
}

func uploadMain(args []string) {

	var wg sync.WaitGroup
	wg.Add(MaxNumOfWorkers)
	jobChannel := make(chan string)
	jobResultChannel := make(chan Files, len(args))

	// start the worker
	for i := 0; i < MaxNumOfWorkers; i++ {
		go worker(&wg, jobChannel, jobResultChannel)
	}

	// send the job
	for _, job := range args {
		jobChannel <- job
	}

	close(jobChannel)
	wg.Wait()
	close(jobResultChannel)

	var jobResults []Files

	for result := range jobResultChannel {
		if result.Error == nil {
			jobResults = append(jobResults, result)
		} else {
			fmt.Printf("%s : %s\n", result.FilePath, result.Error)
		}
	}

	for _, uploads := range jobResults {
		if uploads.UploadResult.Status == http.StatusForbidden {
			fmt.Fprintf(os.Stderr, "Unable to upload %s", uploads.FilePath)
			return
		}

		// Init new list for pretty print
		l := list.NewWriter()

		if originalUrl {
			l.AppendItem("Original URL")
			l.Indent()
			l.AppendItems([]interface{}{uploads.UploadResult.Original.Url})
			l.UnIndent()
		}
		if largeUrl {
			l.AppendItem("Large URL")
			l.Indent()
			l.AppendItems([]interface{}{uploads.UploadResult.Large.Url})
			l.UnIndent()
		}
		if mediumUrl {
			l.AppendItem("Medium URL")
			l.Indent()
			l.AppendItems([]interface{}{uploads.UploadResult.Medium.Url})
			l.UnIndent()
		}
		if thumbUrl {
			l.AppendItem("Thumb URL")
			l.Indent()
			l.AppendItems([]interface{}{uploads.UploadResult.Thumb.Url})
			l.UnIndent()
		}
		if !originalUrl && !largeUrl && !mediumUrl && !thumbUrl {
			l.AppendItem("URL Viewer")
			l.Indent()
			l.AppendItems([]interface{}{uploads.UploadResult.UrlViewer})
			l.UnIndent()
		}
		prettyPrint(uploads.FilePath, l.Render())

	}

}

func worker(wg *sync.WaitGroup, jobChannel <-chan string, resultChannel chan Files) {
	defer wg.Done()

	for job := range jobChannel {
		resultChannel <- ValidateInput(job)
	}
}

func ValidateInput(file string) Files {

	var files Files
	// Create a list of valid image extension
	var validImageExtensions []string = []string{".png", ".jpeg", ".jpg"}

	// Open file to check if dir of file
	fileInfo, err := os.Stat(file)
	// If cannot find file return error
	if err != nil {
		files = Files{Error: err}
		return files
	}

	// If the file is not dir then do some work
	if !fileInfo.IsDir() {
		var validFile bool
		// Check if the file has the correct extension
		for _, extensions := range validImageExtensions {
			if filepath.Ext(file) == extensions {
				validFile = true
			}
		}

		// If it is the correct extension then proceed
		if validFile {
			// Check if you can open the file
			fOpen, err := os.Open(file)
			// If you can open the file throw an error
			if err != nil {
				files = Files{FilePath: file, Error: errors.New("unable to open file")}
				return files
			}
			fOpen.Close()
			// If all is good, continue to upload the file
			uploadResponse := uploadFile(file)

			files = Files{FilePath: file, Type: "file", UploadResult: uploadResponse}

		} else {
			// If file extension is not valid, return error
			files = Files{FilePath: file, Error: errors.New("invalid file type")}
			return files
		}
	} else {
		// If file is not a file, return error
		files = Files{FilePath: file, Error: errors.New("only files are supported")}
		return files
	}
	return files
}

func uploadFile(file string) UploadResponse {
	apiUrl := "https://pstorage.space/api/1/upload"

	f, _ := os.ReadFile(file)

	uploadFile := UploadFile{Filename: file, Key: apiKey, Source: base64.StdEncoding.EncodeToString(f)}
	toJson, _ := json.Marshal(uploadFile)

	resp, _ := http.Post(apiUrl, "application/json", bytes.NewReader(toJson))
	defer resp.Body.Close()
	var uploadResponse UploadResponse
	json.NewDecoder(resp.Body).Decode(&uploadResponse)

	return uploadResponse
}

func prettyPrint(title string, content string) {
	fmt.Printf("%s:\n", title)
	fmt.Println(strings.Repeat("-", len(title)+1))
	for _, line := range strings.Split(content, "\n") {
		fmt.Printf("%s%s\n", "", line)
	}
	fmt.Println()
}
