package simpleuploader

import "os"

type Uploader interface {
	// Upload загружает файл по указанному имени
	Upload(file *os.File, name string) (string, error)
}
