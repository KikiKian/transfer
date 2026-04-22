package main

import (
	"net"
    "os"
    "io"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")

}