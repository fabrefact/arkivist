package main

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestUpload(t *testing.T) {
	w := httptest.NewRecorder()

	file := "some file contents"
	reader := strings.NewReader(file)
	r := httptest.NewRequest("POST", "/media/", reader)
	// required header for streaming uploads to work properly
	r.Header.Add("Upload-Length", strconv.Itoa(len(file)))

	uploadHandler(w, r)

	res := w.Result()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	assert.Equal(t, "Upload successful", string(body))

	// temp solution, need a better way to only delete things created during test
	os.RemoveAll("./uploads")
}
