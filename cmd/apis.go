package cmd

var (
	originalUrl bool
	largeUrl    bool
	mediumUrl   bool
	thumbUrl    bool
)

type Files struct {
	FilePath     string
	Type         string
	Error        error
	UploadResult UploadResponse
}

type UploadResponse struct {
	Status           int        `json:"status"`
	Message          string     `json:"message"`
	OriginalFilename string     `json:"original_filename"`
	Url              string     `json:"url"`
	UrlViewer        string     `json:"url_viewer"`
	Original         ImageStyle `json:"original"`
	Large            ImageStyle `json:"large"`
	Medium           ImageStyle `json:"medium"`
	Thumb            ImageStyle `json:"thumb"`
}

type ImageStyle struct {
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

type UploadFile struct {
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Source   string `json:"source"`
}
