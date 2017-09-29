package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/biz/htmlpdf"
)

func main() {
	html := []byte(`<h1>Hello, World!</h1>`)
	htmlpdf.Init("google-chrome")

	p, err := htmlpdf.Create(html)
	if err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadAll(p)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.Remove(p.Name()); err != nil {
		log.Println("Error removing file:", p.Name())
	}

	fmt.Println(string(b))
}
