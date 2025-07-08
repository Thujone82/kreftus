package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: zipper <output.zip> <file1> <folder1> ...")
		os.Exit(1)
	}

	zipPath := os.Args[1]
	inputPaths := os.Args[2:]

	if err := createZip(zipPath, inputPaths); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating zip: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created %s\n", zipPath)
}

func createZip(zipPath string, inputPaths []string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, inputPath := range inputPaths {
		walkRoot := filepath.Clean(inputPath)

		err := filepath.Walk(walkRoot, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Create a header from the file info
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			// Set the path inside the zip file, making it relative.
			relPath, err := filepath.Rel(filepath.Dir(walkRoot), path)
			if err != nil {
				return err
			}
			if walkRoot == path { // Handle single file case
				relPath = filepath.Base(path)
			}

			header.Name = filepath.ToSlash(relPath)
			if info.IsDir() {
				header.Name += "/" // Mark it as a directory
			}

			// Set the compression method
			header.Method = zip.Deflate

			// Set the proper Unix permissions using the standard library's helper.
			// This is more robust than setting ExternalAttrs manually.
			var perms fs.FileMode
			if info.IsDir() {
				perms = 0755 // rwxr-xr-x for directories
			} else {
				perms = 0644 // rw-r--r-- for regular files
			}
			// Special case for our main executable
			if !info.IsDir() && strings.HasSuffix(header.Name, "vbtc.app/Contents/MacOS/vbtc") {
				perms = 0755 // rwxr-xr-x for the executable
			}
			header.SetMode(perms)

			// Create the entry in the zip file
			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			// If it's a file, copy its contents.
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(writer, file)
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
