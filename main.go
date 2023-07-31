package main

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"os"
	"regexp"
)

var (
	reExtractFileID  = regexp.MustCompile(`([^/]+)/?$`)
	reForwardedHost  = regexp.MustCompile(`host="?([^;"]+)`)
	reForwardedProto = regexp.MustCompile(`proto=(https?)`)
)

// MediaLink contains retrieval information for a given media file
type MediaLink struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// uploadMediaHandler uploads a file to storage
//
// using application/offset+octet-stream requires client to implement the tus resumable upload protocol https://tus.io/protocols/resumable-upload
// otherwise multipart/form-data is assumed
func uploadMediaHandler(composer *tusd.StoreComposer, tusdFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.Header.Get("Content-Type"))

		if r.Header.Get("Content-Type") != "application/offset+octet-stream" {
			handleMultiPartFormUpload(composer, w, r)
			log.Printf("sent multipart form")
			return
		}

		tusdFunc(w, r)

		// return file ID in body and not just location header
		// custom tus client can use this body for functions outside tus spec
		location := w.Header().Get("Location")
		if location != "" {
			// TODO: get provided file name for tus uploads. From metadata?

			// ID can be extracted from URL
			var id string
			if re := reExtractFileID.FindStringSubmatch(location); len(re) == 2 {
				id = re[1]
			}

			link := MediaLink{
				ID:  id,
				URL: location,
			}

			// even though there will only ever be one link, make array for consistency across upload types
			var resp []MediaLink
			resp = append(resp, link)

			w.Header().Set("Content-Type", "application/json")

			err := json.NewEncoder(w).Encode(resp)
			if err != nil {
				// log error I guess
				return
			}
		}
	}
}

// handleMultiPartFormUpload parses multipart/form-data content and attempts to save each file
// using the configured tusd backend.
//
// janky implementation based on generic upload-handling tutorials. Technically *works* but seems sus
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

	// Response body contains information on each uploaded file
	var resp []MediaLink

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

		// set metadata from any headers included in form-data subpart
		meta := make(map[string]string)

		contentType := fileHeader.Header.Get("Content-Type")
		if contentType != "" {
			meta["filetype"] = contentType
		}

		// faking out tusd to store file
		info := tusd.FileInfo{
			Size:     fileHeader.Size,
			IsFinal:  true,
			MetaData: meta,
		}

		// create the file
		upload, err := composer.Core.NewUpload(ctx, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// actually write the file contents
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

		// on successful file upload, generate media information for response
		info, err = upload.GetInfo(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id := info.ID
		url := absFileURL(r, id)

		link := MediaLink{
			ID:       id,
			Filename: fileHeader.Filename,
			URL:      url,
		}
		resp = append(resp, link)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		// log error I guess
		return
	}
}

// Make an absolute URLs to the given upload id.
// Uses host and protocol from the request to build
//
// Modified from tusd unrouted_handler.go
func absFileURL(r *http.Request, id string) string {

	// Read origin and protocol from request
	host, proto := getHostAndProtocol(r, true)

	url := proto + "://" + host + r.URL.Path + id

	return url
}

// getHostAndProtocol extracts the host and used protocol (either HTTP or HTTPS)
// from the given request. If `allowForwarded` is set, the X-Forwarded-Host,
// X-Forwarded-Proto and Forwarded headers will also be checked to
// support proxies.
// Copied from tusd unrouted_handler.go
func getHostAndProtocol(r *http.Request, allowForwarded bool) (host, proto string) {
	if r.TLS != nil {
		proto = "https"
	} else {
		proto = "http"
	}

	host = r.Host

	if !allowForwarded {
		return
	}

	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}

	if h := r.Header.Get("X-Forwarded-Proto"); h == "http" || h == "https" {
		proto = h
	}

	if h := r.Header.Get("Forwarded"); h != "" {
		if re := reForwardedHost.FindStringSubmatch(h); len(re) == 2 {
			host = re[1]
		}

		if re := reForwardedProto.FindStringSubmatch(h); len(re) == 2 {
			proto = re[1]
		}
	}

	return
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
