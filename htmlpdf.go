package htmlpdf

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// chrome startup args
var args = []string{
	"--headless",
	"--no-sandbox",
	"--disable-background-networking",
	"--disable-extensions",
	"--safebrowsing-disable-auto-update",
	"--disable-sync",
	"--disable-gpu",
	"--disable-default-apps",
	"--no-first-run",
	"--hide-scrollbars",
}

// DefaultPDF is the Default Pdf instance that the public functions of this package use to generate a PDF
var DefaultPDF *PDF

// Init sets up the DefaultPdf instance
func Init(chromePath string) {
	DefaultPDF = &PDF{
		chromePath: chromePath,
		args:       args,
	}
}

type PDF struct {
	chromePath string
	args       []string
}

func (pdf *PDF) Create(html []byte) (*os.File, error) {
	// create a temporary file with the html contents
	tmpfile, err := ioutil.TempFile("", "htmlToPdf")
	if err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error creating temp file")
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(html); err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error writing html contents to temp file")
	}
	if err := tmpfile.Close(); err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error closing temp file")
	}

	// create a temp file for the pdf contents
	tmpPdf, err := ioutil.TempFile("", "tempPDF")
	if err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error creating temp file")
	}

	// Create pdf from html contents
	args := append(args, fmt.Sprintf("--print-to-pdf=%s", tmpPdf.Name()), fmt.Sprintf("file://%s", tmpfile.Name()))

	u, err := user.Current()
	if err != nil {
		fmt.Println("error getting user:", err)
	} else {
		fmt.Printf("user: %+v\n", u)
	}

	fmt.Println("exec command", pdf.chromePath, strings.Join(args, " "))
	cmd := exec.Command(pdf.chromePath, args...)

	// watch for errors so we can kill the process if need be
	seb, err := newSyncbuf(cmd.StderrPipe())
	if err != nil {
		return nil, err
	}
	go func() {
		if err := seb.run(); err != nil {
			fmt.Println("Stderr:", seb.buf.String())
			if cmd.Process != nil {
				cmd.Process.Kill()
			} else {
				fmt.Println("stderr: process is nil")
			}
		}
	}()
	sob, err := newSyncbuf(cmd.StdoutPipe())
	if err != nil {
		return nil, err
	}
	go func() {
		if err := sob.run(); err != nil {
			fmt.Println("Stdout:", seb.buf.String())
			if cmd.Process != nil {
				cmd.Process.Kill()
			} else {
				fmt.Println("stdout: process is nil")
			}
		}
	}()

	// Create PDf
	if cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error starting command")
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return nil, errors.Wrap(err, "htmlpdf: Error waiting for command")
	}

	return tmpPdf, nil
}

func Create(html []byte) (*os.File, error) {
	return DefaultPDF.Create(html)
}

func ServePDF(w http.ResponseWriter, filename string, html []byte) error {
	return DefaultPDF.Serve(w, filename, html)
}

func (pdf PDF) Serve(w http.ResponseWriter, filename string, html []byte) error {
	f, err := pdf.Create(html)
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	Serve(w, filename, f)
	return nil
}

// Serve is a helper function that sets the correct http headers to serve a pdf to a browser
func Serve(w http.ResponseWriter, filename string, pdfBody io.ReadCloser) {
	w.Header().Set("Content-Type", "applicaiton/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	_, err := io.Copy(w, pdfBody)
	if err != nil {
		log.Fatal(err)
	}
}

// syncbuf was created because of an exec bug that causes the exec to hang
// this will watch for 'error' in the std err and if it detects and error it will kill the processes
// issue: https://github.com/golang/go/issues/13155
type syncbuf struct {
	pipe   io.ReadCloser
	buf    bytes.Buffer
	ticker *time.Ticker
	sync.RWMutex
}

func newSyncbuf(rc io.ReadCloser, err error) (*syncbuf, error) {
	if err != nil {
		return nil, err
	}
	sb := &syncbuf{
		ticker: time.NewTicker(time.Millisecond * 5),
		buf:    bytes.Buffer{},
		pipe:   rc,
	}

	return sb, nil
}

func (s *syncbuf) run() error {
	for _ = range s.ticker.C {
		b := make([]byte, 1000)
		s.Lock()
		nn, err := s.pipe.Read(b)
		s.Unlock()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if nn > 0 {
			s.buf.Write(b[:nn])
		}
		if strings.Contains(strings.ToLower(s.buf.String()), "error") {
			return fmt.Errorf("output contains 'error' string")
		}
	}
	return nil
}
