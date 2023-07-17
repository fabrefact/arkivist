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

// uploadMediaHandler uploads a file to storage
// tusd requires client to implement the tus resumable upload protocol https://tus.io/protocols/resumable-upload
// if want to support non-tus uploads (ie multipart/form-data) will have to add logic to this function
func uploadMediaHandler(tusdFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// support non-tus uploads?
		tusdFunc(w, r)
	}
}

// getMediaFileHandler retrieves a media file from storage
// Eventually this will be used to request different sizes or formats but for now just pulls original file using tusd
func getMediaFileHandler(tusdFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// add logic to resize images
		tusdFunc(w, r)
	}
}

// createRouter defines all the routes and middleware for the server
// storage routes use the provided data store
func createRouter(store tusd.DataStore) http.Handler {
	r := chi.NewRouter()

	// look into other useful middleware
	r.Use(middleware.Logger)

	composer := tusd.NewStoreComposer()
	composer.UseCore(store)

	// Create a new HTTP handler for the tusd server by providing a configuration.
	// Using UnroutedHandler so API can contain additional routes not supported by tusd
	// The StoreComposer property must be set to allow the handler to function.
	handler, err := tusd.NewUnroutedHandler(tusd.Config{
		BasePath:      "/media/",
		StoreComposer: composer,
	})
	if err != nil {
		log.Fatalf("Unable to create tusd handler: %s", err.Error())
	}

	// handle media routes
	r.Route("/media", func(r chi.Router) {
		// use default tusd middleware to enforce tusd spec and handle OPTIONS
		r.Use(handler.Middleware)
		r.Post("/", uploadMediaHandler(handler.PostFile))
		r.Route("/{mediaId}", func(r chi.Router) {
			r.Get("/", getMediaFileHandler(handler.GetFile))
			r.Delete("/", handler.DelFile)
			r.Head("/", handler.HeadFile)
			r.Patch("/", handler.PatchFile)
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
