package htmlpdfclient

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/biz/htmlpdf"
)

var client *http.Client

// should be set before CreatePDF is called
var PDFServiceURL = ""

func init() {
	client = &http.Client{
		Timeout: time.Minute * 10,
	}
}

func CreatePDF(html []byte) (io.ReadCloser, error) {
	buf := bytes.NewBuffer(html)
	resp, err := client.Post(PDFServiceURL+"/create-pdf", "text/html", buf)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func ServePDF(w http.ResponseWriter, filename string, html []byte) {
	pdf, err := CreatePDF(html)
	if err != nil {
		panic(err)
	}

	if err := htmlpdf.Serve(w, filename, pdf); err != nil {
		panic(err)
	}
}
