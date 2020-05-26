// +build gosseract

package ocr

import (
	"log"
	"sync"

	"github.com/otiai10/gosseract/v2"

	"github.com/piggy-moe/subtitle/config"
)

const VersionTag = " (built with gosseract)"

var (
	lock   sync.Mutex
	client *gosseract.Client
)

func SetupTesseract() {
	client = gosseract.NewClient()
	client.SetLanguage(config.Value.Tesseract.Langs...)
	client.SetPageSegMode(gosseract.PageSegMode(config.Value.Tesseract.Psm))
}

func StopTesseract() {
	client.Close()
}

func RunTesseract(image []byte) (string, error) {
	lock.Lock()
	defer lock.Unlock()
	if err := client.SetImageFromBytes(image); err != nil {
		log.Print(err)
		return "", err
	}
	out, err := client.Text()
	return out, err
}
