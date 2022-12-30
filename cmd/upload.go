/*
Copyright © 2022 HeavyDinosaur
*/
package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	url2 "net/url"
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

const MaxNumOfWorkers = 3

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
	var jobFileChannel = make(chan string, 3)
	var jobResultChannel = make(chan Files, len(args))
	
	// Start the worker for to validate the files
	for i := 0; i < MaxNumOfWorkers; i++ {
		wg.Add(1)
		go worker(&wg, jobFileChannel, jobResultChannel)
	}
	// Send the jobs to start validating the files
	for _, job := range args {
		jobFileChannel <- job
	}

	// Close the jobFileChannel since we have finished sending the jobs
	close(jobFileChannel)
	// Wait for the valide jobs to complete
	wg.Wait()
	// Close the jobResultChannel since we have recieved all the results from the results
	close(jobResultChannel)

	var uploadResultChannel = make(chan Files, len(args))

	var uploadFileChannel = make(chan Files, len(jobResultChannel))
	// Now start the upload worker
	for i := 0; i < MaxNumOfWorkers; i++ {
		wg.Add(1)
		go uploadWorker(&wg, uploadFileChannel, uploadResultChannel)
	}
	// Send the jobs to the upload worker
	for job := range jobResultChannel {
		uploadFileChannel <- job
	}
	// Close uploadFileChannel as we have sent all the job
	close(uploadFileChannel)
	wg.Wait()
	// Close the uploadResultChannel since we have received all responses
	close(uploadResultChannel)
	var finalResult []Files
	for results := range uploadResultChannel {
		finalResult = append(finalResult, results)
	}

	for _, p := range finalResult {

		var printList = list.NewWriter()

		if originalUrl {
			printList.AppendItem("Original URL")
			printList.Indent()
			printList.AppendItems([]interface{}{urlRenamer(p.UploadResult.Url, "original")})
			printList.UnIndent()
		}
		if largeUrl {
			printList.AppendItem("Large URL")
			printList.Indent()
			printList.AppendItems([]interface{}{urlRenamer(p.UploadResult.Url, "large")})
			printList.UnIndent()
		}
		if mediumUrl {
			printList.AppendItem("Medium URL")
			printList.Indent()
			printList.AppendItems([]interface{}{urlRenamer(p.UploadResult.Url, "medium")})
			printList.UnIndent()
		}
		if thumbUrl {
			printList.AppendItem("Thumb URL")
			printList.Indent()
			printList.AppendItems([]interface{}{urlRenamer(p.UploadResult.Url, "thumb")})
			printList.UnIndent()
		}

		if !originalUrl && !largeUrl && !mediumUrl && !thumbUrl {
			printList.AppendItem("URL Viewer")
			printList.Indent()
			printList.AppendItems([]interface{}{p.UploadResult.UrlViewer})
			printList.UnIndent()
		}
		prettyPrint(p.FilePath, printList.Render())
	}

}

func logError(fileError <-chan error) {
	for err := range fileError {
		fmt.Println(err)
	}
}

func worker(wg *sync.WaitGroup, jobFileChannel <-chan string, jobResultChannel chan Files) {
	defer wg.Done()
	for job := range jobFileChannel {
		jobResultChannel <- validateFile(job)
	}
}

// Worker to upload the if there are no error
func uploadWorker(wg *sync.WaitGroup, uploadFileChannel <-chan Files, uploadResultChannel chan Files) {
	defer wg.Done()
	for job := range uploadFileChannel {
		if job.Error != nil {
			fmt.Fprintf(os.Stderr, "%s\n", job.Error)
			//uploadResultChannel <- Files{FilePath: job.FilePath, Error: job.Error}
		} else {
			//uploadResultChannel <- Files{FilePath: job.FilePath}
			uploadResultChannel <- uploadFile(job.FilePath)
		}
	}
}
func validateFile(file string) Files {
	var validatedFiles Files
	// Create a list of valid image extension
	var validImageExtensions []string = []string{".png", ".jpeg", ".jpg"}
	// Open file to check if dir of file
	fileInfo, err := os.Stat(file)
	// If cannot find file return error
	if err != nil {
		return Files{FilePath: file, Error: err}
	}
	// If provided file is a directory, return error
	if fileInfo.IsDir() {
		return Files{FilePath: file, Error: fmt.Errorf("%s : only files are allowed\n", file)}
		// If it is a file continue
	} else {
		// Check for valid image file extension
		for _, extension := range validImageExtensions {
			// If valid extension continue
			if filepath.Ext(file) == extension {
				// Check if you can open the file, to read
				fOpen, err := os.Open(file)
				// If you cant open file to read, throw error
				if err != nil {
					return Files{FilePath: file, Error: err}
				}
				// Close the file
				fOpen.Close()
				// Indicate file is valid
				return Files{FilePath: file}

			} else {
				// File is invalid, hence return an error
				return Files{FilePath: file, Error: fmt.Errorf("%s : invalid file extension. allowed extensions %s\n", file, validImageExtensions)}

			}
		}
		return validatedFiles
	}
}

func uploadFile(file string) Files {
	apiUrl := "https://pstorage.space/api/1/upload"
	f, _ := os.ReadFile(file)

	uploadFile := UploadFile{Filename: file, Key: apiKey, Source: base64.StdEncoding.EncodeToString(f)}
	toJson, _ := json.Marshal(uploadFile)

	resp, err := http.Post(apiUrl, "application/json", bytes.NewReader(toJson))
	if err != nil {
		return Files{FilePath: file, Error: err}
	}
	defer resp.Body.Close()
	var uploadResponse UploadResponse
	json.NewDecoder(resp.Body).Decode(&uploadResponse)

	return Files{FilePath: file, UploadResult: uploadResponse}
}

func prettyPrint(title string, content string) {
	fmt.Printf("%s:\n", title)
	fmt.Println(strings.Repeat("-", len(title)+1))
	for _, line := range strings.Split(content, "\n") {
		fmt.Printf("%s%s\n", "", line)
	}
	fmt.Println()
}

func urlRenamer(url string, posterSize string) string {
	urlParsed, _ := url2.ParseRequestURI(url)

	return strings.Replace(urlParsed.String(), "original", posterSize, 1)
}
