package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	listenPort = ":8080"        // Port to listen on
	remoteAddr = "localhost:8081" // Address of the other peer
)

func main() {
	var wg sync.WaitGroup

	// Start the file server (serving requests)
	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer()
	}()

	// Allow the peer to send file requests
	wg.Add(1)
	go func() {
		defer wg.Done()
		startClient()
	}()

	wg.Wait()
}

// Server part: Handle incoming file requests
func startServer() {
	listener, err := net.Listen("tcp", listenPort)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Listening for incoming requests on", listenPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}
		go handleRequest(conn)
	}
}

// Handle file request from peer
func handleRequest(conn net.Conn) {
	defer conn.Close()

	// Read the requested file name
	request, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}
	request = strings.TrimSpace(request)

	// Open the requested file
	file, err := os.Open(request)
	if err != nil {
		fmt.Fprintf(conn, "Error: %s\n", err.Error())
		fmt.Println("Requested file not found:", request)
		return
	}
	defer file.Close()

	// Send the file content back to the requester
	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("Error sending file:", err)
		return
	}

	fmt.Println("File sent:", request)
}

// Client part: Request files from the other peer
func startClient() {
	for {
		fmt.Println("Enter the name of the file to request from peer:")
		reader := bufio.NewReader(os.Stdin)
		fileName, _ := reader.ReadString('\n')
		fileName = strings.TrimSpace(fileName)

		// Connect to the remote peer
		conn, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			fmt.Println("Error connecting to remote peer:", err)
			return
		}

		// Send the file request
		_, err = conn.Write([]byte(fileName + "\n"))
		if err != nil {
			fmt.Println("Error sending request:", err)
			conn.Close()
			continue
		}

		// Receive the file or an error message
		receivedFile, err := os.Create("received_" + fileName)
		if err != nil {
			fmt.Println("Error creating file:", err)
			conn.Close()
			continue
		}

		// Copy the received data into a file
		_, err = io.Copy(receivedFile, conn)
		if err != nil {
			fmt.Println("Error receiving file:", err)
			receivedFile.Close()
			conn.Close()
			continue
		}

		fmt.Println("File received successfully:", fileName)
		receivedFile.Close()
		conn.Close()
	}
}
