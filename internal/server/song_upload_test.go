package server

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func multipartSongRequest(t *testing.T, data []byte, clipStart string) *http.Request {
	return multipartSongRequestWithTitle(t, data, clipStart, "Test song")
}

func multipartSongRequestWithTitle(t *testing.T, data []byte, clipStart, title string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{
		"title": title, "artist": "Test artist", "genre": "Test genre", "clipStartSeconds": clipStart,
	} {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("song", "input.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/songs", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestReceiveSongUploadRejectsInvalidText(t *testing.T) {
	data := testMP3Bytes(t)
	for _, tc := range []struct {
		name  string
		title string
	}{
		{name: "NUL", title: "bad\x00title"},
		{name: "invalid UTF-8", title: string([]byte{0xff})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MEDIA_ROOT", t.TempDir())
			_, status, err := receiveSongUpload(httptest.NewRecorder(), multipartSongRequestWithTitle(t, data, "0", tc.title))
			if status != http.StatusBadRequest || err == nil {
				t.Fatalf("invalid title = status %d, error %v; want 400", status, err)
			}
		})
	}
}

func TestReceiveSongUploadRejectsInvalidAudioAndRemovesFile(t *testing.T) {
	mediaRoot := t.TempDir()
	t.Setenv("MEDIA_ROOT", mediaRoot)
	_, status, err := receiveSongUpload(httptest.NewRecorder(), multipartSongRequest(t, []byte("not an mp3"), "0"))
	if status != http.StatusBadRequest || err == nil {
		t.Fatalf("invalid audio = status %d, error %v; want 400", status, err)
	}
	files, err := os.ReadDir(filepath.Join(mediaRoot, "incoming"))
	if err != nil || len(files) != 0 {
		t.Fatalf("invalid upload left files: %v, error %v", files, err)
	}
}

func TestReceiveSongUploadRejectsOversizedFile(t *testing.T) {
	t.Setenv("MEDIA_ROOT", t.TempDir())
	largeFile := bytes.Repeat([]byte{'x'}, maxSongFileBytes+1)
	_, status, err := receiveSongUpload(httptest.NewRecorder(), multipartSongRequest(t, largeFile, "0"))
	if status != http.StatusRequestEntityTooLarge || err == nil {
		t.Fatalf("oversized file = status %d, error %v; want 413", status, err)
	}
}

func TestReceiveSongUploadRejectsClipBeyondDuration(t *testing.T) {
	mediaRoot := t.TempDir()
	t.Setenv("MEDIA_ROOT", mediaRoot)
	_, status, err := receiveSongUpload(httptest.NewRecorder(), multipartSongRequest(t, testMP3Bytes(t), "2"))
	if status != http.StatusBadRequest || err == nil {
		t.Fatalf("short remaining clip = status %d, error %v; want 400", status, err)
	}
	files, err := os.ReadDir(filepath.Join(mediaRoot, "incoming"))
	if err != nil || len(files) != 0 {
		t.Fatalf("rejected clip left files: %v, error %v", files, err)
	}
}

func TestReceiveSongUploadSavesValidMP3(t *testing.T) {
	mediaRoot := t.TempDir()
	t.Setenv("MEDIA_ROOT", mediaRoot)
	upload, status, err := receiveSongUpload(httptest.NewRecorder(), multipartSongRequest(t, testMP3Bytes(t), "1"))
	if err != nil || status != 0 {
		t.Fatalf("valid MP3 = status %d, error %v", status, err)
	}
	defer os.Remove(upload.Path)
	if upload.DurationSeconds < 12 || upload.ClipStartSeconds != 1 || upload.Title != "Test song" {
		t.Fatalf("unexpected upload metadata: %+v", upload)
	}
	if _, err := os.Stat(upload.Path); err != nil {
		t.Fatalf("saved upload missing: %v", err)
	}
}

func testMP3Bytes(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required to make the test MP3")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is required to inspect the test MP3")
	}
	fixture := filepath.Join(t.TempDir(), "tone.mp3")
	output, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=13", "-codec:a", "libmp3lame", "-b:a", "64k", "-y", fixture).CombinedOutput()
	if err != nil {
		t.Fatalf("make test MP3: %v: %s", err, output)
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type uploadPart struct {
	name, filename, value string
}

func orderedSongRequest(t *testing.T, parts []uploadPart) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, p := range parts {
		var err error
		if p.filename != "" {
			var part io.Writer
			if part, err = writer.CreateFormFile(p.name, p.filename); err == nil {
				_, err = io.WriteString(part, p.value)
			}
		} else {
			err = writer.WriteField(p.name, p.value)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/songs", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestReceiveSongUploadParsesPartsInAnyOrder(t *testing.T) {
	t.Setenv("MEDIA_ROOT", t.TempDir())
	upload, status, err := receiveSongUpload(httptest.NewRecorder(), orderedSongRequest(t, []uploadPart{
		{name: "song", filename: "input.mp3", value: string(testMP3Bytes(t))},
		{name: "title", value: " Test song "}, {name: "artist", value: "Test artist"},
		{name: "genre", value: "Test genre"}, {name: "clipStartSeconds", value: "1"},
		{name: "ignored", value: "extra text fields are ignored"},
	}))
	if err != nil || status != 0 {
		t.Fatalf("file before fields = status %d, error %v", status, err)
	}
	defer os.Remove(upload.Path)
	if upload.Title != "Test song" {
		t.Fatalf("title=%q", upload.Title)
	}
}

func TestReceiveSongUploadStreamsFileIntoMediaInput(t *testing.T) {
	mediaRoot, tempDir := t.TempDir(), t.TempDir()
	t.Setenv("MEDIA_ROOT", mediaRoot)
	t.Setenv("TMPDIR", tempDir)
	const sent = 2 << 20
	body, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	request := httptest.NewRequest(http.MethodPost, "/songs", body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	go func() {
		part, err := form.CreateFormFile("song", "input.mp3")
		if err == nil {
			_, err = part.Write(bytes.Repeat([]byte{'x'}, sent))
		}
		if err != nil {
			writer.CloseWithError(err)
		}
	}()
	done := make(chan int)
	go func() {
		_, status, _ := receiveSongUpload(httptest.NewRecorder(), request)
		done <- status
	}()

	incoming := filepath.Join(mediaRoot, "incoming")
	deadline := time.Now().Add(5 * time.Second)
	for {
		files, _ := os.ReadDir(incoming)
		if len(files) == 1 {
			if info, err := files[0].Info(); err == nil && info.Size() >= sent-64<<10 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("file data was not streamed into the media input")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if files, err := os.ReadDir(tempDir); err != nil || len(files) != 0 {
		t.Fatalf("upload buffered to temporary files: %v, error %v", files, err)
	}
	writer.CloseWithError(errors.New("client went away"))
	if status := <-done; status != http.StatusBadRequest {
		t.Fatalf("interrupted upload status=%d, want 400", status)
	}
	if files, err := os.ReadDir(incoming); err != nil || len(files) != 0 {
		t.Fatalf("interrupted upload left files: %v, error %v", files, err)
	}
}

func TestReceiveSongUploadRejectsMalformedParts(t *testing.T) {
	fields := []uploadPart{
		{name: "title", value: "Test song"}, {name: "artist", value: "Test artist"},
		{name: "genre", value: "Test genre"}, {name: "clipStartSeconds", value: "0"},
	}
	song := uploadPart{name: "song", filename: "input.mp3", value: "not checked"}
	with := func(extra ...uploadPart) []uploadPart {
		return append(append([]uploadPart{}, fields...), extra...)
	}
	for _, tc := range []struct {
		name   string
		parts  []uploadPart
		status int
	}{
		{"missing file", with(), http.StatusBadRequest},
		{"song as text", with(uploadPart{name: "song", value: "text"}), http.StatusBadRequest},
		{"empty file", with(uploadPart{name: "song", filename: "input.mp3"}), http.StatusBadRequest},
		{"two files", with(song, song), http.StatusBadRequest},
		{"other file field", with(song, uploadPart{name: "cover", filename: "cover.png", value: "x"}), http.StatusBadRequest},
		{"duplicate field", with(song, uploadPart{name: "title", value: "Again"}), http.StatusBadRequest},
		{"missing field", append([]uploadPart{song}, fields[1:]...), http.StatusBadRequest},
		{"oversized field", append([]uploadPart{song, {name: "title", value: strings.Repeat("a", maxSongFieldBytes+1)}}, fields[1:]...), http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mediaRoot := t.TempDir()
			t.Setenv("MEDIA_ROOT", mediaRoot)
			_, status, err := receiveSongUpload(httptest.NewRecorder(), orderedSongRequest(t, tc.parts))
			if status != tc.status || err == nil {
				t.Fatalf("status %d, error %v; want %d", status, err, tc.status)
			}
			if files, err := os.ReadDir(filepath.Join(mediaRoot, "incoming")); err == nil && len(files) != 0 {
				t.Fatalf("rejected upload left files: %v", files)
			}
		})
	}
}

func TestReceiveSongUploadRejectsOversizedBody(t *testing.T) {
	t.Setenv("MEDIA_ROOT", t.TempDir())
	request := orderedSongRequest(t, []uploadPart{{name: "padding", value: strings.Repeat("x", maxSongBodyBytes)}})
	_, status, err := receiveSongUpload(httptest.NewRecorder(), request)
	if status != http.StatusRequestEntityTooLarge || err == nil {
		t.Fatalf("oversized body = status %d, error %v; want 413", status, err)
	}
}

func validSongParts(t *testing.T) []uploadPart {
	return []uploadPart{
		{name: "title", value: "Test song"}, {name: "artist", value: "Test artist"},
		{name: "genre", value: "Test genre"}, {name: "clipStartSeconds", value: "0"},
		{name: "song", filename: "input.mp3", value: string(testMP3Bytes(t))},
	}
}

func TestReceiveSongUploadLimitsPartCount(t *testing.T) {
	for _, tc := range []struct {
		parts  int
		status int
	}{{maxSongParts, 0}, {maxSongParts + 1, http.StatusBadRequest}} {
		t.Run(strconv.Itoa(tc.parts), func(t *testing.T) {
			t.Setenv("MEDIA_ROOT", t.TempDir())
			parts := validSongParts(t)
			for len(parts) < tc.parts {
				parts = append(parts, uploadPart{name: "ignored", value: "x"})
			}
			upload, status, err := receiveSongUpload(httptest.NewRecorder(), orderedSongRequest(t, parts))
			if status != tc.status || (err == nil) != (tc.status == 0) {
				t.Fatalf("status %d, error %v; want %d", status, err, tc.status)
			}
			if upload.Path != "" {
				os.Remove(upload.Path)
			}
		})
	}
}

func TestReceiveSongUploadSkipsNamelessParts(t *testing.T) {
	t.Setenv("MEDIA_ROOT", t.TempDir())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, disposition := range []string{`form-data; filename="other.mp3"`, `form-data`} {
		part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Disposition": {disposition}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, "skipped"); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range validSongParts(t) {
		var err error
		if p.filename != "" {
			var part io.Writer
			if part, err = writer.CreateFormFile(p.name, p.filename); err == nil {
				_, err = io.WriteString(part, p.value)
			}
		} else {
			err = writer.WriteField(p.name, p.value)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/songs", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	upload, status, err := receiveSongUpload(httptest.NewRecorder(), request)
	if err != nil || status != 0 {
		t.Fatalf("nameless parts = status %d, error %v", status, err)
	}
	os.Remove(upload.Path)
}
