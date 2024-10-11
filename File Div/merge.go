package main

import (
	"fmt"
	"io"
	"os"
)

func mergeChunks(outputFileName string, totalParts int) error {
	// Create the output file where we will merge all chunks
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Iterate through all parts and merge them into the output file
	for partNum := 1; partNum <= totalParts; partNum++ {
		// Generate the part file name
		partFileName := fmt.Sprintf("chunk_%d", partNum)

		// Open the part file
		partFile, err := os.Open(partFileName)
		if err != nil {
			return fmt.Errorf("failed to open part file %s: %v", partFileName, err)
		}

		// Copy the part file contents to the output file
		_, err = io.Copy(outputFile, partFile)
		if err != nil {
			partFile.Close()
			return fmt.Errorf("failed to copy part file %s: %v", partFileName, err)
		}

		// Close the part file after copying
		partFile.Close()
	}

	return nil
}

func displayFile(filePath string) error {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Read and display the file contents
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %v", err)
		}
		if n == 0 {
			break
		}
		fmt.Print(string(buffer[:n]))
	}

	return nil
}

func main() {
	totalParts := 16                 // Change this to the number of parts
	outputFileName := "merged.txt"  // The file to save the merged chunks

	// Merge all chunks into one file
	err := mergeChunks(outputFileName, totalParts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display the contents of the merged file
	fmt.Println("Merged file contents:")
	err = displayFile(outputFileName)
	if err != nil {
		fmt.Printf("Error displaying file: %v\n", err)
	}
}
