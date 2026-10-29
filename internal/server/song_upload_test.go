package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
