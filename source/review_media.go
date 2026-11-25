package main

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    "cloud.google.com/go/storage"
)

var reviewImageName = regexp.MustCompile(`^[0-9a-f]{64}\.jpg$`)

func (h *AdminHandler) UploadReviewImage(w http.ResponseWriter,r *http.Request) {
    if r.Method!=http.MethodPost {w.WriteHeader(405);return}
    if !isAdminAuthenticated(h.cfg,r){http.Error(w,"Access denied",403);return}
    file,header,err:=r.FormFile("photo")
    if err!=nil {http.Error(w,"Выберите фото",400);return}
    defer file.Close()
    if header.Size>20<<20 {http.Error(w,"Максимальный размер — 20 МБ",413);return}
    dir,err:=os.MkdirTemp("","review-image-")
    if err!=nil{http.Error(w,"Temporary storage unavailable",500);return}
    defer os.RemoveAll(dir)
    input:=filepath.Join(dir,"original")
    raw,err:=os.Create(input)
    if err!=nil{http.Error(w,"Temporary storage unavailable",500);return}
    _,copyErr:=io.Copy(raw,file)
    closeErr:=raw.Close()
    if copyErr!=nil || closeErr!=nil {http.Error(w,"Upload failed",400);return}
    ctx,cancel:=context.WithTimeout(r.Context(),30*time.Second)
    defer cancel()
    output:=filepath.Join(dir,"photo.jpg")
    cmd:=exec.CommandContext(ctx,"vips","thumbnail",input,output+"[Q=78,keep=none]","1200","--height","900","--size","down")
    if err=cmd.Run();err!=nil{http.Error(w,"Не удалось прочитать фотографию. Проверьте формат и содержимое файла.",422);return}
    data,err:=os.ReadFile(output)
    if err!=nil{http.Error(w,"Image conversion failed",500);return}
    name:=fmt.Sprintf("%x.jpg",sha256.Sum256(data))
    client,err:=storage.NewClient(ctx)
    if err!=nil{http.Error(w,"Storage unavailable",503);return}
    defer client.Close()
    if err=uploadLocalFileToGCS(ctx,client.Bucket(h.cfg.GCSBucketName),output,"review-media/"+name,"image/jpeg");err!=nil{
        http.Error(w,"Не удалось сохранить фотографию",503);return
    }
    w.Header().Set("Content-Type","application/json")
    json.NewEncoder(w).Encode(map[string]string{"url":"/review-media/"+name})
}
func serveReviewImage(cfg Config,w http.ResponseWriter,r *http.Request){
    if r.Method!="GET" && r.Method!="HEAD"{w.WriteHeader(405);return}
    name:=strings.TrimPrefix(r.URL.Path,"/review-media/")
    if !reviewImageName.MatchString(name){http.NotFound(w,r);return}
    client,err:=storage.NewClient(r.Context())
    if err!=nil{http.Error(w,"Storage unavailable",503);return}
    defer client.Close()
    reader,err:=client.Bucket(cfg.GCSBucketName).Object("review-media/"+name).NewReader(r.Context())
    if errors.Is(err,storage.ErrObjectNotExist){http.NotFound(w,r);return}
    if err!=nil{http.Error(w,"Storage unavailable",503);return}
    defer reader.Close()
    w.Header().Set("Cache-Control","public,max-age=31536000,immutable")
    w.Header().Set("Content-Type","image/jpeg")
    w.Header().Set("X-Content-Type-Options","nosniff")
    if r.Method=="GET"{io.Copy(w,reader)}
}
