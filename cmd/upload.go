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
	"net/http"
	url2 "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:          "upload",
	Short:        "Upload images",
	Example:      "pstorage upload file dir/files dir/* --thumb",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
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

	go logError(errorChannel)
	// Start the validateWorker for to validate the files
	for i := 0; i < MaxNumOfWorkers; i++ {
		wg.Add(1)
		go validateWorker(&wg, jobFileChannel, jobResultChannel)
	}
	// Send the jobs to start validating the files
	for _, job := range args {
		jobFileChannel <- job
	}

	// Close the jobFileChannel since we have finished sending the jobs
	close(jobFileChannel)
	// Wait for validate jobs to complete
	wg.Wait()
	// Close the jobResultChannel since we have recieved all the uploadedResults from the uploadedResults
	close(jobResultChannel)

	var uploadResultChannel = make(chan Files, len(args))

	var uploadFileChannel = make(chan Files, len(jobResultChannel))
	// Now start the upload validateWorker
	for i := 0; i < MaxNumOfWorkers; i++ {
		wg.Add(1)
		go uploadWorker(&wg, uploadFileChannel, uploadResultChannel)
	}
	// Send the jobs to the upload validateWorker
	for job := range jobResultChannel {
		uploadFileChannel <- job
	}
	// Close uploadFileChannel as we have sent all the job
	close(uploadFileChannel)
	wg.Wait()
	// Close the uploadResultChannel since we have received all responses
	close(uploadResultChannel)
	close(errorChannel)
	
	for uploadedResults := range uploadResultChannel {
		if uploadedResults != (Files{}) {
			var printList = list.NewWriter()
			if originalUrl {
				printList.AppendItem("Original URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(uploadedResults.UploadResult.Url, "original")})
				printList.UnIndent()
			}
			if largeUrl {
				printList.AppendItem("Large URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(uploadedResults.UploadResult.Url, "large")})
				printList.UnIndent()
			}
			if mediumUrl {
				printList.AppendItem("Medium URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(uploadedResults.UploadResult.Url, "medium")})
				printList.UnIndent()
			}
			if thumbUrl {
				printList.AppendItem("Thumb URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(uploadedResults.UploadResult.Url, "thumb")})
				printList.UnIndent()
			}

			if !originalUrl && !largeUrl && !mediumUrl && !thumbUrl {
				printList.AppendItem("URL Viewer")
				printList.Indent()
				printList.AppendItems([]interface{}{uploadedResults.UploadResult.UrlViewer})
				printList.UnIndent()
			}
			prettyPrint(uploadedResults.FilePath, printList.Render())
		}
	}

}

func validateWorker(wg *sync.WaitGroup, jobFileChannel <-chan string, jobResultChannel chan Files) {
	defer wg.Done()
	for job := range jobFileChannel {
		jobResultChannel <- validateFile(job)
	}
}

// Worker to upload the if there are no error
func uploadWorker(wg *sync.WaitGroup, uploadFileChannel <-chan Files, uploadResultChannel chan Files) {
	defer wg.Done()
	for job := range uploadFileChannel {
		if job != (Files{}) {
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
		errorChannel <- err

		return Files{}
	}
	// If provided file is a directory, return error
	if fileInfo.IsDir() {
		errorChannel <- fmt.Errorf("%s : only files are allowed", file)
		return Files{}

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
					errorChannel <- err
					return Files{}
				}
				// Close the file
				fOpen.Close()
				// Indicate file is valid
				return Files{FilePath: file}

			} else {
				// File is invalid, hence return an error
				errorChannel <- fmt.Errorf("%s : invalid file extension. allowed extensions %s", file, validImageExtensions)
				return Files{}
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
		errorChannel <- err
		return Files{}
	}

	defer resp.Body.Close()
	var uploadResponse UploadResponse
	json.NewDecoder(resp.Body).Decode(&uploadResponse)

	if uploadResponse.Status != http.StatusOK {
		errorChannel <- fmt.Errorf("error uploading %s : %s", file, uploadResponse.Message)
		return Files{}
	}
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
