package main

import (
	"fmt"
	"net/http"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"log"
	"io"
	"flag"
	"sort"
	"github.com/unrolled/render"
)

const maxBucketItems = 10000

var (
	s3Bucket = flag.String(
		"s3.bucket", "",
		"S3 bucket name",
	)
	s3Host = flag.String(
		"s3.host", "",
		"S3 host",
	)
	s3AccessKey = flag.String(
		"s3.access.key", "",
		"S3 access key",
	)
	s3ScecretKey = flag.String(
		"s3.secret.key", "",
		"S3 secret key",
	)
	listenAddress = flag.String(
		"web.listen-address", ":8080",
		"Address to listen on",
	)
)

type application struct {
	bucket *s3.Bucket
	render *render.Render
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if len(r.URL.Path[1:]) == 0 {
		// show listing
		entries := app.listBucket()
		app.render.HTML(w, http.StatusOK, "listing", entries)
	} else {
		// download from S3
		app.download(r.URL.Path[1:], w)
	}
}

func (app *application) listBucket() (entries []string) {

	resp, err := app.bucket.List("", "", "", maxBucketItems)
	if err != nil {
		fmt.Printf("error listing bucket contents: %v", err)
		return
	}

	for _, key := range resp.Contents {
		entries = append(entries, key.Key)
	}
	sort.Strings(entries)
	return
}

func (app *application) download (path string, w http.ResponseWriter) {
	rc, err := app.bucket.GetReader(path)
	if err != nil {
		fmt.Fprintf(w, "Got error while reading from S3: %v", err)
		return
	}

	io.Copy(w, rc)
	defer rc.Close()
}

func main() {
	flag.Parse()

	fmt.Printf("starting on port %s", *listenAddress)

	app := &application{
		bucket: newBucket(),
		render: render.New(render.Options{
			Layout: "layout",
		}),
	}

	http.ListenAndServe(*listenAddress, app)
}

func newBucket() (*s3.Bucket) {
	auth, err := aws.GetAuth(*s3AccessKey, *s3ScecretKey)
	if err != nil {
		log.Fatal(err)
	}
	region := aws.Region{S3Endpoint: *s3Host}

	client := &s3.S3{
		Auth:   auth,
		Region: region,
		HTTPClient: func() *http.Client {
			return http.DefaultClient
		},
	}

	return &s3.Bucket{S3: client, Name: *s3Bucket}
}
