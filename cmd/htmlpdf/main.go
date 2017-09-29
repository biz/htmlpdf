package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/biz/htmlpdf"
	"github.com/coreos/go-systemd/daemon"
)

var (
	port       string
	chromePath string
)

func init() {
	flag.StringVar(&port, "port", "80", "port that the http server binds to")
	flag.StringVar(&chromePath, "chrome-path", "google-chrome", "Path to Google Chrome")
}

func main() {
	flag.Parse()
	htmlpdf.Init(chromePath)

	mux := http.NewServeMux()

	mux.HandleFunc("/create-pdf", createPDF)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Minute * 10, // the created PDF's could be large, need lots of time
		Handler:      mux,
		Addr:         ":" + port,
	}

	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatal(err)
	}

	daemon.SdNotify(false, "READY=1")
	go health(mux)
	runtime.Gosched()

	if err := server.Serve(l); err != nil {
		log.Fatal(err)
	}
}

func createPDF(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	f, err := htmlpdf.Create(b)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer os.Remove(f.Name())

	io.Copy(w, f)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`ok`))
}

func health(mux *http.ServeMux) {
	interval, err := daemon.SdWatchdogEnabled(false)
	if err != nil || interval == 0 {
		return
	}
	for {
		_, err := http.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			daemon.SdNotify(false, "WATCHDOG=1")
		}
		time.Sleep(interval / 3)
	}
}
