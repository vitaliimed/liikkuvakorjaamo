package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func saveUploadedPhotosToGCS(cfg Config, r *http.Request) ([]string, []string, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil, nil
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		return nil, nil, nil
	}

	if len(files) > maxPhotoCount {
		return nil, nil, fmt.Errorf("please upload up to %d photos", maxPhotoCount)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create storage client: %w", err)
	}
	defer client.Close()

	bucket := client.Bucket(cfg.GCSBucketName)

	prefix := strings.Trim(strings.TrimSpace(cfg.GCSPhotoPrefix), "/")
	if prefix == "" {
		prefix = "lead-photos"
	}

	tempDir, err := os.MkdirTemp("", "lead-photos-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}

	var refs []string
	var tempPaths []string

	cleanupTempOnError := true
	defer func() {
		if cleanupTempOnError {
			deleteTempFiles(tempPaths)
			_ = os.Remove(tempDir)
		}
	}()

	for i, header := range files {
		if header == nil || header.Filename == "" {
			continue
		}

		if header.Size > maxPhotoSize {
			return nil, nil, fmt.Errorf("each photo must be smaller than 10 MB")
		}

		file, err := header.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open uploaded file: %w", err)
		}

		sniff := make([]byte, 512)
		n, _ := file.Read(sniff)
		if n == 0 {
			_ = file.Close()
			return nil, nil, fmt.Errorf("empty file is not allowed")
		}

		detectedType := http.DetectContentType(sniff[:n])

		if seeker, ok := file.(io.Seeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}

		if !strings.HasPrefix(detectedType, "image/") {
			_ = file.Close()
			return nil, nil, fmt.Errorf("only image files are allowed")
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			if exts, _ := mime.ExtensionsByType(detectedType); len(exts) > 0 {
				ext = exts[0]
			}
		}
		if ext == "" {
			ext = ".jpg"
		}

		stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
		localName := fmt.Sprintf("%s_%02d%s", stamp, i+1, ext)
		tempPath := filepath.Join(tempDir, localName)

		dst, err := os.Create(tempPath)
		if err != nil {
			_ = file.Close()
			return nil, nil, fmt.Errorf("create temp file: %w", err)
		}

		if _, err := io.Copy(dst, file); err != nil {
			_ = dst.Close()
			_ = file.Close()
			return nil, nil, fmt.Errorf("save temp file: %w", err)
		}

		_ = dst.Close()
		_ = file.Close()

		objectName := prefix + "/" + localName

		if err := uploadLocalFileToGCS(ctx, bucket, tempPath, objectName, detectedType); err != nil {
			return nil, nil, err
		}

		refs = append(refs, "gs://"+cfg.GCSBucketName+"/"+objectName)
		tempPaths = append(tempPaths, tempPath)
	}

	if err := cleanupOldGCSPhotos(ctx, bucket, prefix, cfg.GCSMaxPhotos); err != nil {
		log.Printf("GCS cleanup warning: %v", err)
	}

	cleanupTempOnError = false
	return refs, tempPaths, nil
}

func uploadLocalFileToGCS(ctx context.Context, bucket *storage.BucketHandle, localPath, objectName, contentType string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open temp file for gcs upload: %w", err)
	}
	defer file.Close()

	writer := bucket.Object(objectName).NewWriter(ctx)
	writer.ContentType = contentType

	if _, err := io.Copy(writer, file); err != nil {
		_ = writer.Close()
		return fmt.Errorf("upload to gcs: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close gcs writer: %w", err)
	}

	return nil
}

func cleanupOldGCSPhotos(ctx context.Context, bucket *storage.BucketHandle, prefix string, maxPhotos int) error {
	if maxPhotos <= 0 {
		return nil
	}

	query := &storage.Query{
		Prefix: strings.Trim(prefix, "/") + "/",
	}

	it := bucket.Objects(ctx, query)

	var objectNames []string

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list gcs objects: %w", err)
		}

		if attrs == nil || strings.TrimSpace(attrs.Name) == "" {
			continue
		}

		objectNames = append(objectNames, attrs.Name)
	}

	if len(objectNames) <= maxPhotos {
		return nil
	}

	extra := len(objectNames) - maxPhotos

	for i := 0; i < extra; i++ {
		if err := bucket.Object(objectNames[i]).Delete(ctx); err != nil {
			return fmt.Errorf("delete old gcs object %s: %w", objectNames[i], err)
		}
	}

	return nil
}

func deleteTempFiles(paths []string) {
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return
	}

	tempRoot = tempRoot + string(os.PathSeparator)

	seenDirs := map[string]bool{}

	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(absPath, tempRoot) {
			continue
		}

		_ = os.Remove(absPath)

		dir := filepath.Dir(absPath)
		if !seenDirs[dir] {
			seenDirs[dir] = true
			_ = os.Remove(dir)
		}
	}
}
