package main

import (
	"cloud.google.com/go/storage"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func liikkuvaAdminPhoto(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !isAdminAuthenticated(cfg, r) {
		http.Error(w, "Войдите в админку", 401)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", 405)
		return
	}
	index, e := strconv.Atoi(r.URL.Query().Get("index"))
	if e != nil || index < 0 || index > 2 {
		http.Error(w, "Некорректный номер фото", 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	record, e := getLeadRecord(ctx, cfg, r.URL.Query().Get("lead"))
	if e != nil || index >= len(record.Lead.PhotoURLs) {
		http.NotFound(w, r)
		return
	}
	ref := record.Lead.PhotoURLs[index]
	if !strings.HasPrefix(ref, "gs://") {
		http.NotFound(w, r)
		return
	}
	bucket, object, e := splitGCSRef(ref)
	if e != nil || bucket != cfg.GCSBucketName || !strings.HasPrefix(object, strings.Trim(cfg.GCSPhotoPrefix, "/")+"/") {
		http.Error(w, "Forbidden", 403)
		return
	}
	client, e := storage.NewClient(ctx)
	if e != nil {
		http.Error(w, "Фото временно недоступно", 503)
		return
	}
	defer client.Close()
	reader, e := client.Bucket(bucket).Object(object).NewReader(ctx)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	defer reader.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(reader, head)
	kind := http.DetectContentType(head[:n])
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		w.Header().Set("Content-Disposition", "attachment; filename=photo")
		kind = "application/octet-stream"
	}
	w.Header().Set("Content-Type", kind)
	w.Write(head[:n])
	io.Copy(w, io.LimitReader(reader, 10<<20))
}
