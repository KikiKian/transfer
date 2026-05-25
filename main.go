package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

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

func compress(folder, password string) {
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
		return nil
	})

	zipWriter.Flush()
}

func decompress(outputFolder, password string) {
	compressed, err := zip.OpenReader("./main.zip")
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

func send(serverAdress, zipPath string) {
	conn, err := net.Dial("tcp", serverAdress)
	if err != nil {
		log.Fatalln("error dialing server", err)
	}
	defer conn.Close()

	zipFile, err := os.Open(zipPath)
	if err != nil {
		log.Fatalln("error opening zip on send func", err)
	}
	defer zipFile.Close()

	io.Copy(conn, zipFile)
}

func handleConnection(conn net.Conn, outputFolder, password string) {
	defer conn.Close()

	zipFile, err := os.Create("./main.zip")
	if err != nil {
		log.Fatalln("error creating zip in handleConnection", err)
	}
	defer zipFile.Close()

	io.Copy(zipFile, conn)
	zipFile.Close()

	decompress(outputFolder, password)
}

func main() {
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/read", handleRead)
	http.ListenAndServe(":3030", nil)
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	folder := r.FormValue("folder")
	password := r.FormValue("password")
	serverAddr := r.FormValue("serverAddr")

	compress(folder, password)
	send(serverAddr, "./main.zip")
	w.Write([]byte("transfer complete!"))

}

func handleRead(w http.ResponseWriter, r *http.Request) {
	outputFolder := r.FormValue("outputFolder")
	password := r.FormValue("password")

	go openPort(outputFolder, password)
	w.Write([]byte("receiving... files will be saved to " + outputFolder))
}
