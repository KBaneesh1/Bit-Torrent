package main

import (
	"fmt"
	"io"
	"os"
)

const chunkSize = 16  // 16KB

func splitFile(filePath string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, chunkSize)
	partNum := 1

	for {
		// Read into the buffer
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %v", err)
		}

		// Stop if we've reached the end of the file
		if n == 0 {
			break
		}

		// Create a new chunk file
		chunkFileName := fmt.Sprintf("chunk_%d", partNum)
		chunkFile, err := os.Create(chunkFileName)
		if err != nil {
			return fmt.Errorf("failed to create chunk file: %v", err)
		}

		// Write the chunk
		_, err = chunkFile.Write(buffer[:n])
		if err != nil {
			chunkFile.Close()
			return fmt.Errorf("failed to write to chunk file: %v", err)
		}

		chunkFile.Close()
		partNum++
	}

	return nil
}

func main() {
	filePath := "check.txt"
	err := splitFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("File split into chunks successfully.")
	}
}
