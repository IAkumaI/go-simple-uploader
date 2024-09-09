package simpleuploader

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAkumaI/retry"
	"github.com/jlaffaye/ftp"
)

type FTPUploader struct {
	host       string
	login      string
	password   string
	pathPrefix string
	dir        string
	urlPrefix  string
}

// NewFTP создает настроенный FTPUploader(s) uploader
func NewFTP(addr string, login string, password string, dir string, pathPrefix string, urlPrefix string) Uploader {
	return &FTPUploader{
		host:       addr,
		login:      login,
		password:   password,
		dir:        dir,
		pathPrefix: pathPrefix,
		urlPrefix:  urlPrefix,
	}
}

func (uploader *FTPUploader) Upload(file *os.File, name string) (string, error) {
	result := ""

	storPath := uploader.pathPrefix + "/" + name
	if uploader.dir != "" {
		storPath = strings.TrimRight(uploader.dir, "/") + "/" + strings.TrimLeft(storPath, "/")
	}

	err := retry.Do(10, func(retryCount int) error {
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			log.Println("Can not seek to file start")
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		client, err := ftp.Dial(uploader.host, ftp.DialWithTimeout(20*time.Second))
		if err != nil {
			log.Printf("Не удается подключиться к FTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}
		defer client.Quit()

		err = client.Login(uploader.login, uploader.password)
		if err != nil {
			log.Printf("Не удается авторизоваться на FTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		// Создаем директорию
		storDirs := strings.Split(filepath.Dir(storPath), "/")
		for i := 0; i <= len(storDirs); i++ {
			testPath := strings.Join(storDirs[:i], "/")
			if testPath != "" {
				entries, err := client.List(testPath)
				if len(entries) == 0 || err != nil { // Директория не существует
					err := client.MakeDir(testPath)
					if err != nil {
						log.Printf("[SKIP] Ошибка при создании директории %s: %v\n", testPath, err)
					}
				}
			}
		}

		err = client.Stor(storPath, file)
		if err != nil {
			log.Printf("Не удается загрузить файл на FTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		result = uploader.urlPrefix + uploader.pathPrefix + "/" + name
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
