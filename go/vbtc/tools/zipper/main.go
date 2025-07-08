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

			// Determine the relative path for the file inside the zip.
			relPath, err := filepath.Rel(filepath.Dir(walkRoot), path)
			if err != nil {
				return err
			}
			// Handle the case where we are zipping a single file, not a directory.
			if walkRoot == path {
				relPath = filepath.Base(path)
			}

			// Manually create the header to ensure it has Unix attributes,
			// which is critical for macOS to respect executable permissions.
			// Using zip.FileInfoHeader() can inherit MS-DOS attributes on Windows.
			header := &zip.FileHeader{
				Name:     filepath.ToSlash(relPath),
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}

			// Set the appropriate file mode.
			if info.IsDir() {
				header.Name += "/"
				// Set directory permissions. fs.ModeDir is crucial.
				header.SetMode(0755 | fs.ModeDir)
			} else {
				// Set standard file permissions.
				perms := fs.FileMode(0644)
				// Explicitly set executable permissions for the main binary.
				if strings.HasSuffix(header.Name, "vbtc.app/Contents/MacOS/vbtc") {
					perms = 0755
				}
				header.SetMode(perms)
			}

			// Create the entry in the zip file.
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
