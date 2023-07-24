package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
)

// uploadMediaHandler uploads a file to storage
// tusd requires client to implement the tus resumable upload protocol https://tus.io/protocols/resumable-upload
// if want to support non-tus uploads (ie multipart/form-data) will have to add logic to this function
func uploadMediaHandler(composer *tusd.StoreComposer, tusdFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// support non-tus uploads?
		log.Print(r.Header.Get("Content-Type"))

		if r.Header.Get("Content-Type") != "application/offset+octet-stream" {
			handleMultiPartFormUpload(composer, w, r)
			log.Printf("sent multipart form")
			return
		}

		tusdFunc(w, r)
	}
}

// janky implementation to support regular multipart form uploads. Technically *works* but seems sus
func handleMultiPartFormUpload(composer *tusd.StoreComposer, w http.ResponseWriter, r *http.Request) {
	// 32 MB is the default used by FormFile() and the example I found on the internet
	// Request body up to this much will be read into memory, the rest into temporary files on disk
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get a reference to the fileHeaders.
	// They are accessible only after ParseMultipartForm is called
	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {

		// Open each file
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// this isn't great for memory management, but every example does this
		defer file.Close()

		ctx := context.Background()

		// faking out tusd to store file
		info := tusd.FileInfo{
			Size:    fileHeader.Size,
			IsFinal: true,
		}

		upload, err := composer.Core.NewUpload(ctx, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bytesWritten, err := upload.WriteChunk(ctx, 0, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Wrote %d bytes", bytesWritten)

		if err := upload.FinishUpload(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
	// the problem with this is not sending back any of the filenames of the created files
	w.WriteHeader(http.StatusCreated)
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
	log.Print("testing")

	// handle media routes
	r.Route("/media", func(r chi.Router) {
		// use default tusd middleware to enforce tusd spec and handle OPTIONS
		r.Use(TempMiddleware)
		r.Use(handler.Middleware)
		r.Post("/", uploadMediaHandler(composer, handler.PostFile))
		r.Route("/{mediaId}", func(r chi.Router) {
			r.Get("/", getMediaFileHandler(handler.GetFile))
			r.Delete("/", handler.DelFile)
			r.Head("/", handler.HeadFile)
			r.Patch("/", handler.PatchFile)
		})
	})

	return r
}

func TempMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// fake out this header so I can test until I decide how to properly handle this
		r.Header.Set("Tus-Resumable", "1.0.0")

		// Proceed with routing the request
		h.ServeHTTP(w, r)
	})
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
