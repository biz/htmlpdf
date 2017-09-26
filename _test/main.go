package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/biz/htmlpdf"
)

func main() {
	html := []byte(`<h1>Hello, World!</h1>`)
	htmlpdf.Init(`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := htmlpdf.ServePDF(w, "test.pdf", html)
		if err != nil {
			fmt.Fprintf(w, "%+v", err)
			return
		}
	})

	if err := http.ListenAndServe(":9000", nil); err != nil {
		log.Fatal(err)
	}
}
