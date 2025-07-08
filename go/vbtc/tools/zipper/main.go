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
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(filepath.Dir(walkRoot), path)
			if err != nil {
				return err
			}
			if walkRoot == path { // Handle single file case
				relPath = filepath.Base(path)
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = filepath.ToSlash(relPath)

			// Manually set Unix permissions to ensure they are respected on macOS.
			// The standard zip library sets the creator OS to MS-DOS, which can
			// cause macOS to ignore the executable bits.
			var perms fs.FileMode = 0644
			if strings.HasSuffix(header.Name, "vbtc.app/Contents/MacOS/vbtc") {
				perms = 0755
			}
			header.CreatorVersion = 3 << 8                          // Set creator OS to Unix
			header.ExternalAttrs = (uint32(perms) | 0o100000) << 16 // Set file mode and type
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}
