package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"io"
	"log"
	"net/http/httptest"
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

// "Files" uploaded by this test are actually empty;
// unsure for now if this is a problem with the test or a problem with the code
func TestUpload(t *testing.T) {
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	assert.Equal(t, 201, res.StatusCode)
	assert.Equal(t, "Upload successful", string(body))
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "some test data", string(body))
}
