package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"log"
	"net/http"
	"os"
)

// writeResponse abstracts away writing response bodies entirely because I don't like the syntax
func writeResponse(w http.ResponseWriter, body string) {
	_, err := w.Write([]byte(body))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// upload a file to storage
func uploadMediaHandler(store tusd.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// tusd only seems to work for single file uploads, only handle one file at a time

		// A storage backend for tusd may consist of multiple different parts which
		// handle upload creation, locking, termination and so on. The composer is a
		// place where all those separated pieces are joined together. In this example
		// we only use the file store but you may plug in multiple.
		composer := tusd.NewStoreComposer()
		composer.UseCore(store)

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

		// probably don't need to return a response body at all but for now it's useful
		writeResponse(w, "Upload successful")
	}
}

// retrieve a media file from storage
func getMediaFileHandler(store tusd.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, "hi")
	}
}

// delete a media file from storage
func deleteMediaFileHandler(store tusd.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, "hi")
	}
}

// createRouter defines all the routes and middleware for the server
// storage routes use the provided data store
func createRouter(store tusd.DataStore) http.Handler {
	r := chi.NewRouter()

	// look into other useful middleware
	r.Use(middleware.Logger)

	// handle media routes
	r.Route("/media", func(r chi.Router) {
		r.Post("/", uploadMediaHandler(store))
		r.Route("/{mediaId}", func(r chi.Router) {
			r.Get("/", getMediaFileHandler(store))
			r.Delete("/", deleteMediaFileHandler(store))
		})
	})

	return r
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Need to figure out what configurations are required for S3store
	// store := s3store.S3Store{}

	// to allow testing for now, lets just save to local directory
	err := os.MkdirAll("./uploads", 0777)
	if err != nil {
		log.Fatalf("Error creating media directory: %s", err.Error())
	}
	store := filestore.FileStore{
		Path: "./uploads",
	}

	router := createRouter(store)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Unexpected error: %s", err.Error())
	}
}
