package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexmullins/zip"
)

func addFileToZip(zipWriter *zip.Writer, filePath, nameInZip, password string) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("could not open file %s: %v", filePath, err)
	}
	defer f.Close()

	w, err := zipWriter.Encrypt(nameInZip, password)
	if err != nil {
		log.Fatalf("could not encrypt %s in zip: %v", nameInZip, err)
	}

	if _, err := io.Copy(w, f); err != nil {
		log.Fatalf("could not write %s to zip: %v", nameInZip, err)
	}
}

var numFiles int
var totalFiles int

func compress(folder, password string) {
	// Count total files first so we can report progress as done/total
	totalFiles = 0
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		totalFiles++
		return nil
	})

	numFiles = 0
	outputZip, err := os.Create("./main.zip")
	if err != nil {
		log.Fatalln("could not create zip file:", err)
	}
	defer outputZip.Close()

	zipWriter := zip.NewWriter(outputZip)
	defer zipWriter.Close()

	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		nameInZip, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}

		addFileToZip(zipWriter, path, nameInZip, password)
		numFiles++
		return nil
	})

	zipWriter.Flush()
}

func decompress(zipPath, outputFolder, password string) {
	compressed, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Fatalln("error decompressing zip", err)
	}
	defer compressed.Close()

	for _, file := range compressed.File {
		file.SetPassword(password)

		destPath := filepath.Join(outputFolder, file.Name)

		os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
		bytes, err := file.Open()

		if err != nil {
			log.Fatalln("error opening file", err)
		}

		destination, err := os.Create(destPath)
		if err != nil {
			log.Fatalln("error creating destination path", err)
		}

		io.Copy(destination, bytes)

		destination.Close()
		bytes.Close()
	}
}

func openPort(outputFolder, password string) {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("error listening on port", err)
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		log.Fatalln("error accepting connection", err)
	}
	handleConnection(conn, outputFolder, password)
}

func send(serverAdress, zipPath string) error {
	conn, err := net.Dial("tcp", serverAdress)
	if err != nil {
		return fmt.Errorf("could not connect to server: %w", err)
	}
	defer conn.Close()

	zipFile, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("could not open zip: %w", err)
	}
	defer zipFile.Close()

	io.Copy(conn, zipFile)
	return nil
}

func handleConnection(conn net.Conn, outputFolder, password string) {
	defer conn.Close()

	zipFile, err := os.CreateTemp("", "received-*.zip")
	if err != nil {
		log.Fatalln("error creating temp zip in handleConnection", err)
	}
	zipPath := zipFile.Name()
	defer os.Remove(zipPath)

	io.Copy(zipFile, conn)
	zipFile.Close()

	decompress(zipPath, outputFolder, password)
}

func main() {
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/read", handleRead)
	http.HandleFunc("/progress", handleProgress)
	http.HandleFunc("/localip", handleLocalIP)
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.ListenAndServe(":3030", nil)
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	folder := r.FormValue("folder")
	password := r.FormValue("password")
	serverAddr := r.FormValue("serverAddr")

	compress(folder, password)
	if err := send(serverAddr, "./main.zip"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("transfer complete!"))
}

func handleRead(w http.ResponseWriter, r *http.Request) {
	outputFolder := r.FormValue("outputFolder")
	password := r.FormValue("password")

	go openPort(outputFolder, password)
	w.Write([]byte("receiving... files will be saved to " + outputFolder))
}

func handleLocalIP(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		w.Write([]byte("unknown"))
		return
	}
	defer conn.Close()
	ip := conn.LocalAddr().(*net.UDPAddr).IP.String()
	w.Write([]byte(ip))
}

func handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	flusher := w.(http.Flusher)
	for numFiles < totalFiles || totalFiles == 0 {
		fmt.Fprintf(w, "data: %d/%d\n\n", numFiles, totalFiles)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}
	// Send final 100%
	fmt.Fprintf(w, "data: %d/%d\n\n", totalFiles, totalFiles)
	flusher.Flush()
}
