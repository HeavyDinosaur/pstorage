/*
Copyright © 2022 HeavyDinosaur
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
	"sync"
)

// urlCmd represents the url command
var urlCmd = &cobra.Command{
	Use:          "url",
	Short:        "upload images from a url",
	Example:      "pstorage url https://url.com/image.jpeg https://url2.com/image.png",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		urlUpload(args)
	},
}

func urlUpload(urls []string) {
	var wg sync.WaitGroup
	var urlJobChannel = make(chan string, MaxNumOfWorkers)
	var urlResultChannel = make(chan Files, len(urls))

	// Start the errorLogger in a separate goroutine
	go logError(errorChannel)
	// Start the urlWorker
	for i := 0; i < MaxNumOfWorkers; i++ {
		wg.Add(1)
		go urlWorker(&wg, urlJobChannel, urlResultChannel)
	}

	// Send jobs to the urlJobChannel
	for _, job := range urls {
		validateUrl, err := url.ParseRequestURI(job)
		// If error validate url skip it
		if err != nil {
			// TODO: use errors.is for custom error message
			errorChannel <- err
			continue
		}
		// If good send to urlJobChannel
		urlJobChannel <- validateUrl.String()
	}
	// Close the urlJobChannel since we have sent all urls
	close(urlJobChannel)
	wg.Wait()
	close(urlResultChannel)
	close(errorChannel)

	for urlResults := range urlResultChannel {
		if urlResults != (Files{}) {
			var printList = list.NewWriter()
			if originalUrl {
				printList.AppendItem("Original URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(urlResults.UploadResult.Url, "original")})
				printList.UnIndent()
			}
			if largeUrl {
				printList.AppendItem("Large URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(urlResults.UploadResult.Url, "large")})
				printList.UnIndent()
			}
			if mediumUrl {
				printList.AppendItem("Medium URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(urlResults.UploadResult.Url, "medium")})
				printList.UnIndent()
			}
			if thumbUrl {
				printList.AppendItem("Thumb URL")
				printList.Indent()
				printList.AppendItems([]interface{}{urlRenamer(urlResults.UploadResult.Url, "thumb")})
				printList.UnIndent()
			}

			if !originalUrl && !largeUrl && !mediumUrl && !thumbUrl {
				printList.AppendItem("URL Viewer")
				printList.Indent()
				printList.AppendItems([]interface{}{urlResults.UploadResult.UrlViewer})
				printList.UnIndent()
			}
			prettyPrint(urlResults.FilePath, printList.Render())
		}
	}
}

func urlWorker(wg *sync.WaitGroup, urlJobChannel <-chan string, urlResultChannel chan Files) {
	defer wg.Done()
	for job := range urlJobChannel {
		urlResultChannel <- uploadUrl(job)
	}

}

func uploadUrl(uploadUrl string) Files {

	apiUrl, _ := url.Parse("https://pstorage.space/api/1/upload")

	// Build the url
	query := apiUrl.Query()
	query.Set("key", apiKey)
	query.Set("source", uploadUrl)
	apiUrl.RawQuery = query.Encode()

	// Send the request
	resp, err := http.Get(apiUrl.String())
	if err != nil {
		errorChannel <- err
		return Files{}
	}
	defer resp.Body.Close()
	var uploadResponse UploadResponse
	// Save the response to uploadResponse
	json.NewDecoder(resp.Body).Decode(&uploadResponse)

	// If status is not 200, return error
	if uploadResponse.Status != http.StatusOK {
		errorChannel <- fmt.Errorf("error uploading %s : %s", uploadUrl, uploadResponse.Message)
		return Files{}
	}

	return Files{FilePath: uploadUrl, UploadResult: uploadResponse}

}

func init() {
	rootCmd.AddCommand(urlCmd)
	urlCmd.Flags().BoolVar(&originalUrl, "original", false, "Print original url")
	urlCmd.Flags().BoolVar(&largeUrl, "large", false, "Print large url")
	urlCmd.Flags().BoolVar(&mediumUrl, "medium", false, "Print medium url")
	urlCmd.Flags().BoolVar(&thumbUrl, "thumb", false, "Print thumb url")
}
