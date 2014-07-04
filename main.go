package main

import (
	"flag"
	"fmt"
	"github.com/nfnt/resize"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var AppPath string

func main() {
	var err error
	var port = flag.Int("port", 80, "http listen and serve the port,default=80")
	flag.Parse()
	if AppPath, err = filepath.Abs("."); err != nil {
		panic(err)
	}
	http.Handle("/get/", http.StripPrefix("/get/", http.FileServer(http.Dir(filepath.Join(AppPath, "fs")))))
	http.Handle("/upload/", http.StripPrefix("/upload/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			log.Printf("error at upload only post")
			w.Write([]byte("error,only post"))
			return
		}
		file, _, err := r.FormFile("upload")
		if err != nil {
			log.Printf("error at upload fromfile:%s", err)
			w.Write([]byte("r.FormFile has a error :" + err.Error()))
			return
		}
		defer file.Close()
		fileName := filepath.Join(AppPath, "fs", r.URL.Path)
		err = os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
		if err != nil {
			log.Printf("error at upload MkdirAll:%s", err)
			w.Write([]byte("MkdirAll has a error :" + err.Error()))
			return
		}

		var diskFile *os.File
		diskFile, err = os.Create(fileName)
		if err != nil {
			log.Printf("error at upload create:%s", err)
			w.Write([]byte("create has a error :" + err.Error()))
			return
		}
		defer diskFile.Close()
		_, err = io.Copy(diskFile, file)
		diskFile.Close()
		if err != nil {
			log.Printf("error at upload copy:%s", err)
			w.Write([]byte("copy has a error :" + err.Error()))
			return
		}
		log.Printf("upload %s", fileName)
		if strings.ToLower(filepath.Ext(fileName)) == ".jpg" {
			//build small picture
			if err := buildSmallJpeg(fileName[:len(fileName)-4]+"_small.jpg", fileName); err != nil {
				log.Printf("error at build small jpg:%s", err)
				w.Write([]byte("build small jpg error :" + err.Error()))
				return

			}
			w.Write([]byte(fmt.Sprintf("<html><body style='font-size:12px;'>Success upload the file:<a href='%s' target='_blank' >Picture</a>&nbsp;&nbsp;and auto build the <a href='%s' target='_blank'>Small Picture</a>",
				"/get/"+r.URL.Path,
				"/get/"+r.URL.Path[:len(r.URL.Path)-4]+"_small.jpg")))
		}
	})))
	http.Handle("/exists/", http.StripPrefix("/exists/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Write([]byte("error,only get"))
			return
		}
		exists := true
		if _, err := os.Stat(filepath.Join(AppPath, "fs", r.URL.Path)); err != nil {
			exists = false
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(r.URL.Query().Get("callback") + "("))
		w.Write([]byte(fmt.Sprintf(`{"exists":%v}`, exists)))
		w.Write([]byte(");"))
		return
	})))
	http.Handle("/delete/", http.StripPrefix("/delete/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := true
		var err error
		if r.Method != "GET" {
			ok = false
		}
		if err = os.Remove(filepath.Join(AppPath, "fs", r.URL.Path)); err != nil {
			log.Printf("error at delete:%s", err)
			ok = false
		}
		if strings.ToLower(filepath.Ext(r.URL.Path)) == ".jpg" {
			if err = os.Remove(filepath.Join(AppPath, "fs", r.URL.Path[:len(r.URL.Path)-4]+"_small.jpg")); err != nil {
				log.Printf("error at delete small file:%s", err)
				ok = false
			}
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(r.URL.Query().Get("callback") + "("))
		w.Write([]byte(fmt.Sprintf(`{"ok":%v,"err":%q}`, ok, err)))
		w.Write([]byte(");"))
		return
	})))
	fmt.Printf("start http://localhost:%d...\n", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
func buildSmallJpeg(destFile, srcFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()
	srcJPG, err := jpeg.Decode(src)
	if err != nil {
		return err
	}
	destImage := resize.Thumbnail(720, 900, srcJPG, resize.Bilinear)

	destF, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer destF.Close()

	err = jpeg.Encode(destF, destImage, &jpeg.Options{90})
	if err != nil {
		return err
	}
	return nil
}