package simpleuploader

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

import (
	"github.com/IAkumaI/retry"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPUploader struct {
	host       string
	login      string
	password   string
	pathPrefix string
	dir        string
	urlPrefix  string
}

// NewSFTP создает настроенный SFTPUploader(s) uploader
func NewSFTP(addr string, login string, password string, dir string, pathPrefix string, urlPrefix string) Uploader {
	return &SFTPUploader{
		host:       addr,
		login:      login,
		password:   password,
		dir:        dir,
		pathPrefix: pathPrefix,
		urlPrefix:  urlPrefix,
	}
}

func (uploader *SFTPUploader) Upload(file *os.File, name string) (string, error) {
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

		config := &ssh.ClientConfig{
			User:            uploader.login,
			Auth:            []ssh.AuthMethod{ssh.Password(uploader.password)},
			Timeout:         30 * time.Second,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		conn, err := ssh.Dial("tcp", uploader.host, config)
		if err != nil {
			log.Printf("Не удается подключиться к SFTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		client, err := sftp.NewClient(conn)
		if err != nil {
			log.Printf("Не удается создать клиент SFTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}
		defer client.Close()

		err = client.MkdirAll(filepath.Dir(storPath))
		if err != nil {
			log.Printf("Не удается создать директорию SFTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		time.Sleep(50 * time.Millisecond)
		dstFile, err := client.OpenFile(storPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
		if err != nil {
			log.Printf("Не удается создать файл на SFTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}
		defer dstFile.Close()

		time.Sleep(50 * time.Millisecond)
		bytes, err := io.Copy(dstFile, file)
		if err != nil {
			log.Printf("Не удается записать файл на SFTP(%d): %v\n", retryCount, err)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return err
		}

		if bytes == 0 {
			log.Printf("Не удается записать файл (0 байт) на SFTP(%d)\n", retryCount)
			log.Printf("Retry in 5 sec... (%d)\n", retryCount)
			time.Sleep(time.Second * 5)
			return errors.New("Zero bytes written")
		}

		result = uploader.urlPrefix + uploader.pathPrefix + "/" + name
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
