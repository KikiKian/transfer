package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexmullins/zip"
)

type LogEntry struct {
	Percent int   `json:"percent"`
	Time    int64 `json:"time"`
}

var transferLog []LogEntry
var transferLogMutex sync.Mutex

var numFiles int
var totalFiles int

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

func handleConnection(conn net.Conn, outputFolder, password string) {
	defer conn.Close()

	zipFile, err := os.CreateTemp("", "received-*.zip")
	if err != nil {
		log.Fatalln("error creating temp zip", err)
	}
	defer os.Remove(zipFile.Name())

	io.Copy(zipFile, conn)
	zipFile.Close()

	decompress(zipFile.Name(), outputFolder, password)
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
			log.Fatalln("error creating destination", err)
		}

		io.Copy(destination, bytes)

		destination.Close()
		bytes.Close()
	}
}

func compress(folder, password string) {
	totalFiles = 0

	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalFiles++
		}
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

		nameInZip, _ := filepath.Rel(folder, path)
		addFileToZip(zipWriter, path, nameInZip, password)

		numFiles++

		percent := 0
		if totalFiles > 0 {
			percent = (numFiles * 100) / totalFiles
		}

		transferLogMutex.Lock()
		transferLog = append(transferLog, LogEntry{
			Percent: percent,
			Time:    time.Now().Unix(),
		})
		transferLogMutex.Unlock()

		return nil
	})

	zipWriter.Flush()
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	serverAddr := r.FormValue("serverAddr")

	transferLogMutex.Lock()
	transferLog = []LogEntry{}
	transferLogMutex.Unlock()

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	outputZip, err := os.Create("./main.zip")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer outputZip.Close()

	zipWriter := zip.NewWriter(outputZip)
	defer zipWriter.Close()

	totalFiles = 0
	for _, files := range r.MultipartForm.File {
		totalFiles += len(files)
	}

	numFiles = 0

	for _, files := range r.MultipartForm.File {
		for _, fh := range files {

			file, err := fh.Open()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			writer, err := zipWriter.Encrypt(fh.Filename, password)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			io.Copy(writer, file)
			file.Close()

			numFiles++

			percent := 0
			if totalFiles > 0 {
				percent = (numFiles * 100) / totalFiles
			}

			transferLogMutex.Lock()
			transferLog = append(transferLog, LogEntry{
				Percent: percent,
				Time:    time.Now().Unix(),
			})
			transferLogMutex.Unlock()
		}
	}

	zipWriter.Close()
	outputZip.Close()

	if err := send(serverAddr, "./main.zip"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte("transfer complete!"))
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	transferLogMutex.Lock()
	defer transferLogMutex.Unlock()

	json.NewEncoder(w).Encode(transferLog)
}

func handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	flusher := w.(http.Flusher)
	last := 0

	for numFiles < totalFiles || totalFiles == 0 {

		percent := 0
		if totalFiles > 0 {
			percent = (numFiles * 100) / totalFiles
		}

		if percent != last {
			transferLogMutex.Lock()
			transferLog = append(transferLog, LogEntry{
				Percent: percent,
				Time:    time.Now().Unix(),
			})
			transferLogMutex.Unlock()

			last = percent
		}

		fmt.Fprintf(w, "data: %d/%d\n\n", numFiles, totalFiles)
		flusher.Flush()

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: %d/%d\n\n", totalFiles, totalFiles)
	flusher.Flush()
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

func main() {
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/read", handleRead)
	http.HandleFunc("/progress", handleProgress)
	http.HandleFunc("/localip", handleLocalIP)
	http.HandleFunc("/log", handleLog)

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.ListenAndServe(":3030", nil)
}
