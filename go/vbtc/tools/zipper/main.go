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

			// Determine the relative path for the file inside the zip archive.
			relPath, err := filepath.Rel(filepath.Dir(walkRoot), path)
			if err != nil {
				return err
			}
			// Handle the case where we are zipping a single file instead of a directory.
			if walkRoot == path {
				relPath = filepath.Base(path)
			}

			// Manually create the header to have full control over attributes.
			// This is crucial for cross-platform compatibility, especially for macOS.
			header := &zip.FileHeader{
				Name:     filepath.ToSlash(relPath),
				Method:   zip.Deflate,
				Modified: info.ModTime(),
			}

			// Set the creator OS to Unix (3) and encode the file permissions in the
			// external attributes. This is the most reliable way to ensure that
			// tools on macOS (like the default Archive Utility) respect the permissions.
			header.CreatorVersion = 3 << 8 // Set creator OS to Unix
			if info.IsDir() {
				header.Name += "/"
				// Set directory permissions: drwxr-xr-x
				header.ExternalAttrs = (0o755 | 0o40000) << 16
			} else {
				// Set standard file permissions: -rw-r--r--
				perms := uint32(0o644)
				// Explicitly set executable permissions for the main binary: -rwxr-xr-x
				if strings.HasSuffix(header.Name, "vbtc.app/Contents/MacOS/vbtc") {
					perms = 0o755
				}
				header.ExternalAttrs = (perms | 0o100000) << 16
			}

			// Create the entry in the zip file and write the file data if it's not a directory.
			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

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
