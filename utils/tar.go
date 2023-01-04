package utils

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Tar(src string, of *os.File) error {

	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	//of, err := os.Create(outFile)
	//if err != nil {
	//	return fmt.Errorf("Could not create tarball file '%s', got error '%s'", outFile, err.Error())
	//}
	//defer of.Close()

	tw := tar.NewWriter(of)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		if err != nil {
			fmt.Printf("Generic error for %v: %v\n", fi, err)
			return err
		}

		// skip non-regular files
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			fmt.Printf("Cannot create file header for %v\n", fi)
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		var strippedSrc string
		if filepath.Dir(src) == "." && !strings.HasPrefix(src, ".") {
			strippedSrc = src // nothing to do
		} else {
			strippedSrc = strings.Replace(file, filepath.Dir(src), "", -1)
		}

		header.Name = strings.TrimPrefix(strippedSrc, string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			fmt.Printf("Cannot write file header for %v\n", fi)
			return err
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Cannot open file %v\n", fi)
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			fmt.Printf("Cannot write file %v\n", fi)
			return err
		}

		f.Close()

		return nil
	})
}
