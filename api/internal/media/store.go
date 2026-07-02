package media

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"

	"github.com/xuroi/xuroi/api/internal/ids"
)

const (
	MaxUploadBytes = 12 << 20 // 12 MB
	maxEdge        = 4096
	thumbMaxEdge   = 520
	webpQuality    = 86
	thumbQuality   = 78
)

var mediaNameRe = regexp.MustCompile(`^med_[0-9a-z]{26}(_thumb)?\.webp$`)

type UploadResult struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bytes    int    `json:"bytes"`
}

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		dir = filepath.Join("..", "infra", "uploads")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: abs}, nil
}

func ValidMediaName(name string) bool {
	clean := filepath.Base(name)
	return clean == name && mediaNameRe.MatchString(clean)
}

func (s *Store) SaveUpload(r io.Reader) (UploadResult, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return UploadResult{}, err
	}
	if int64(len(raw)) > MaxUploadBytes {
		return UploadResult{}, fmt.Errorf("file too large (max 12MB)")
	}

	img, err := imaging.Decode(bytes.NewReader(raw))
	if err != nil {
		return UploadResult{}, fmt.Errorf("unsupported image format")
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxEdge || h > maxEdge {
		img = imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}

	var src image.Image = img

	id := ids.New("med_")
	filename := id + ".webp"
	path := filepath.Join(s.dir, filename)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, src, &webp.Options{Quality: webpQuality}); err != nil {
		return UploadResult{}, fmt.Errorf("webp encode: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return UploadResult{}, err
	}

	thumbFilename := id + "_thumb.webp"
	if err := s.writeThumb(src, thumbFilename); err != nil {
		return UploadResult{}, err
	}

	return UploadResult{
		ID:       id,
		URL:      "/api/media/" + filename,
		ThumbURL: "/api/media/" + thumbFilename,
		Width:    w,
		Height:   h,
		Bytes:    buf.Len(),
	}, nil
}

func (s *Store) writeThumb(src image.Image, filename string) error {
	thumb := src
	bounds := thumb.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > thumbMaxEdge || h > thumbMaxEdge {
		thumb = imaging.Fit(thumb, thumbMaxEdge, thumbMaxEdge, imaging.Lanczos)
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, thumb, &webp.Options{Quality: thumbQuality}); err != nil {
		return fmt.Errorf("thumb encode: %w", err)
	}
	return os.WriteFile(filepath.Join(s.dir, filename), buf.Bytes(), 0o644)
}

func (s *Store) ensureThumb(fullName string) error {
	if !ValidMediaName(fullName) || strings.Contains(fullName, "_thumb") {
		return os.ErrNotExist
	}

	thumbName := strings.TrimSuffix(fullName, ".webp") + "_thumb.webp"
	thumbPath := filepath.Join(s.dir, thumbName)
	if _, err := os.Stat(thumbPath); err == nil {
		return nil
	}

	fullPath := filepath.Join(s.dir, fullName)
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	img, err := imaging.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	return s.writeThumb(img, thumbName)
}

func (s *Store) Open(name string) (*os.File, error) {
	clean := filepath.Base(name)
	if !ValidMediaName(clean) {
		return nil, os.ErrNotExist
	}

	if strings.Contains(clean, "_thumb") {
		if err := s.ensureThumb(strings.Replace(clean, "_thumb.webp", ".webp", 1)); err != nil {
			return nil, err
		}
	}

	path := filepath.Join(s.dir, clean)
	return os.Open(path)
}

func (s *Store) Dir() string { return s.dir }