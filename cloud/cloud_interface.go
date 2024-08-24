package cloud

type CloudClient interface {
	UploadFile(fileName *string, filePath string) error
	DownloadFile(fileName *string, downloadPath string) error
}
