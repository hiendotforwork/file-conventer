package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	web "file-converter"
	"file-converter/internal/handler"
	"file-converter/internal/service"
)

func main() {
	soffice, err := exec.LookPath("soffice")
	if err != nil {
		log.Fatal("LibreOffice (soffice) is required")
	}
	app := handler.New(service.New(soffice), web.Index(), web.Static())
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: "0.0.0.0:" + port, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	log.Printf("File Converter listening on 0.0.0.0:%s", port)
	log.Fatal(server.ListenAndServe())
}
