package media

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"

	"github.com/xuroi/xuroi/api/internal/ids"
)

const (
	MaxAvatarBytes = 4 << 20 // 4 MB
	avatarFullEdge = 256
	avatarSmEdge   = 64
	avatarQuality  = 88
	avatarSmQual   = 82
	minAvatarEdge  = 64
)

type AvatarResult struct {
	URL   string `json:"url"`
	SmURL string `json:"sm_url"`
	ID    string `json:"id"`
}

func (s *Store) SaveAvatar(r io.Reader) (AvatarResult, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxAvatarBytes+1))
	if err != nil {
		return AvatarResult{}, err
	}
	if int64(len(raw)) > MaxAvatarBytes {
		return AvatarResult{}, fmt.Errorf("file too large (max 4MB)")
	}

	img, err := imaging.Decode(bytes.NewReader(raw))
	if err != nil {
		return AvatarResult{}, fmt.Errorf("unsupported image format")
	}

	sq := squareCrop(img)
	bounds := sq.Bounds()
	if bounds.Dx() < minAvatarEdge || bounds.Dy() < minAvatarEdge {
		return AvatarResult{}, fmt.Errorf("image too small (min %d×%d)", minAvatarEdge, minAvatarEdge)
	}

	id := ids.New("avt_")
	fullName := id + ".webp"
	smName := id + "_sm.webp"

	full := imaging.Fill(sq, avatarFullEdge, avatarFullEdge, imaging.Center, imaging.Lanczos)
	sm := imaging.Fill(sq, avatarSmEdge, avatarSmEdge, imaging.Center, imaging.Lanczos)

	if err := s.writeWebP(full, fullName, avatarQuality); err != nil {
		return AvatarResult{}, err
	}
	if err := s.writeWebP(sm, smName, avatarSmQual); err != nil {
		_ = os.Remove(filepath.Join(s.dir, fullName))
		return AvatarResult{}, err
	}

	base := "/api/media/"
	return AvatarResult{
		ID:    id,
		URL:   base + fullName,
		SmURL: base + smName,
	}, nil
}

func (s *Store) DeleteAvatarFiles(url string) error {
	name := avatarFileFromURL(url)
	if name == "" {
		return nil
	}
	base := strings.TrimSuffix(name, ".webp")
	for _, n := range []string{base + ".webp", base + "_sm.webp"} {
		_ = os.Remove(filepath.Join(s.dir, n))
	}
	return nil
}

func avatarFileFromURL(url string) string {
	const prefix = "/api/media/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	name := filepath.Base(url)
	if !ValidMediaName(name) {
		return ""
	}
	return name
}

func squareCrop(img image.Image) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	side := w
	if h < side {
		side = h
	}
	x0 := bounds.Min.X + (w-side)/2
	y0 := bounds.Min.Y + (h-side)/2
	return imaging.Crop(img, image.Rect(x0, y0, x0+side, y0+side))
}

func (s *Store) writeWebP(img image.Image, filename string, quality float32) error {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: quality}); err != nil {
		return fmt.Errorf("webp encode: %w", err)
	}
	return os.WriteFile(filepath.Join(s.dir, filename), buf.Bytes(), 0o644)
}