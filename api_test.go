package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"io"
	"log"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"testing"
)

// create a simple filestore for storing test files
// creates the requested directory, and returns a cleaner function to delete all directory contents
func setupTestStore(path string) (tusd.DataStore, func()) {
	err := os.MkdirAll(path, 0777)
	cleanupFunc := func() {
		os.RemoveAll(path)
	}
	if err != nil {
		log.Fatalf("Error creating media directory: %s", err.Error())
	}
	store := filestore.FileStore{
		Path: path,
	}
	return store, cleanupFunc
}

func TestTusUpload(t *testing.T) {
	store, cleaner := setupTestStore("./test-uploads")
	defer cleaner()

	router := createRouter(store)

	w := httptest.NewRecorder()

	file := "some file contents"
	reader := strings.NewReader(file)
	r := httptest.NewRequest("POST", "/media/", reader)
	// required headers for resumable uploads to work properly
	r.Header.Add("Upload-Length", strconv.Itoa(len(file)))
	r.Header.Add("Content-Type", "application/offset+octet-stream")
	r.Header.Add("Tus-Resumable", "1.0.0")

	router.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal()
	}

	assert.Equal(t, 201, res.StatusCode)
	// ideally want to assert something meaningful about the response, but ID/URL are random so can't naively compare
	assert.NotNil(t, body)
}

func TestMultipartUpload(t *testing.T) {
	store, cleaner := setupTestStore("./test-uploads")
	defer cleaner()

	router := createRouter(store)

	w := httptest.NewRecorder()

	file := "some file contents"

	var b bytes.Buffer
	wr := multipart.NewWriter(&b)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		`form-data; name="file"; filename="random.txt"`)
	h.Set("Content-Type", "text/plain")
	fw, err := wr.CreatePart(h)
	fw.Write([]byte(file))
	wr.Close()

	r := httptest.NewRequest("POST", "/media/", &b)
	// expected headers for multipart/form-data upload
	r.Header.Add("Content-Type", wr.FormDataContentType())

	router.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal()
	}

	assert.Equal(t, 201, res.StatusCode)
	// ideally want to assert something meaningful about the response, but ID/URL are random so can't naively compare
	assert.NotNil(t, body)
}

func TestDownload(t *testing.T) {
	store := filestore.FileStore{
		Path: "./testdata",
	}

	router := createRouter(store)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/media/testfile/", nil)

	router.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "some test data", string(body))
}
