package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"log"
	"net/http"
	"os"
)

// fetch file from s3
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi"))
}

// upload a file to s3
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// tusd only seems to work for single file uploads, only handle one file at a time

	// Need to figure out what configurations are required for S3store
	// probably shouldn't be created here?
	// store := s3store.S3Store{}

	// to allow testing for now, lets just save to local directory
	err := os.MkdirAll("./uploads", 0777)
	if err != nil {
		log.Fatalf("Error creating media directory: %s", err.Error())
	}
	store := filestore.FileStore{
		Path: "./uploads",
	}

	// A storage backend for tusd may consist of multiple different parts which
	// handle upload creation, locking, termination and so on. The composer is a
	// place where all those separated pieces are joined together. In this example
	// we only use the file store but you may plug in multiple.
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	// Create a new HTTP handler for the tusd server by providing a configuration.
	// The StoreComposer property must be set to allow the handler to function.
	handler, err := tusd.NewUnroutedHandler(tusd.Config{
		BasePath:              "/media/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		log.Fatalf("Unable to create handler: %s", err.Error())
	}

	// need to look into what this does and what is written to the response
	handler.PostFile(w, r)

	// Chi has a terrible return ergonomics, write a helper function for this
	w.Write([]byte(fmt.Sprintf("Upload successful")))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	r := chi.NewRouter()

	// look into other useful middleware
	r.Use(middleware.Logger)

	// who even knows what this is for
	r.Get("/", indexHandler)

	// handle media routes
	r.Route("/media", func(r chi.Router) {
		r.Post("/", uploadHandler)
		r.Route("/{mediaId}", func(r chi.Router) {
			// add get and delete for individual media
		})
	})
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Unexpected error: %s", err.Error())
	}
}
