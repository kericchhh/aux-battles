package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

var ErrOutsideMedia = errors.New("path outside media directory")

func New(root string) Store {
	if root == "" {
		root = "storage"
	}
	return Store{root: root}
}

func FromEnv() Store {
	return New(os.Getenv("MEDIA_ROOT"))
}

func (s Store) dir(name string) (string, error) {
	return filepath.Abs(filepath.Join(s.root, name))
}

func (s Store) CreateInput() (*os.File, error) {
	incoming, err := s.dir("incoming")
	if err != nil {
		return nil, fmt.Errorf("resolve media directory: %w", err)
	}
	if err := os.MkdirAll(incoming, 0750); err != nil {
		return nil, fmt.Errorf("create media directory: %w", err)
	}
	file, err := os.CreateTemp(incoming, "song-*.mp3")
	if err != nil {
		return nil, fmt.Errorf("save uploaded file: %w", err)
	}
	return file, nil
}

func (s Store) InputPath(path string) (string, error) {
	incoming, err := s.dir("incoming")
	if err != nil {
		return "", fmt.Errorf("resolve incoming directory: %w", err)
	}
	if !within(incoming, path) {
		return "", fmt.Errorf("song input is outside incoming media directory")
	}
	realIncoming, err := filepath.EvalSymlinks(incoming)
	if err != nil {
		return "", fmt.Errorf("resolve incoming directory: %w", err)
	}
	realInput, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve song input: %w", err)
	}
	if !within(realIncoming, realInput) {
		return "", fmt.Errorf("song input resolves outside incoming media directory")
	}
	info, err := os.Stat(realInput)
	if err != nil {
		return "", fmt.Errorf("stat song input: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("song input is not a regular file")
	}
	return realInput, nil
}

func (s Store) OpenAudio(path string) (*os.File, os.FileInfo, error) {
	songs, err := s.dir("songs")
	if err != nil {
		return nil, nil, fmt.Errorf("resolve song media directory: %w", err)
	}
	relative, err := filepath.Rel(songs, path)
	if err != nil || !filepath.IsLocal(relative) {
		return nil, nil, fmt.Errorf("song audio path is outside media directory")
	}
	root, err := os.OpenRoot(songs)
	if err != nil {
		return nil, nil, fmt.Errorf("open song media directory: %w", err)
	}
	defer root.Close()
	file, err := root.Open(relative)
	if err != nil {
		return nil, nil, fmt.Errorf("open song audio: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("inspect song audio: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		_ = file.Close()
		return nil, nil, fmt.Errorf("song audio is empty or not a regular file")
	}
	return file, info, nil
}

func (s Store) RemoveInput(path string) error {
	incoming, err := s.dir("incoming")
	if err != nil {
		return err
	}
	if !within(incoming, path) {
		return fmt.Errorf("refusing to remove input: %w", ErrOutsideMedia)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s Store) DiscardOutput(directory string) error {
	if directory == "" {
		return nil
	}
	songs, err := s.dir("songs")
	if err != nil {
		return err
	}
	if !within(songs, directory) {
		return fmt.Errorf("refusing to discard output: %w", ErrOutsideMedia)
	}
	return os.RemoveAll(directory)
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && filepath.IsLocal(relative)
}
