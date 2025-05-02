package processor

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func ExtractTarGz(tarPath string) (manifest, nodeCSV string, err error) {
	file, err := os.Open(tarPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}

		switch filepath.Base(header.Name) {
		case "manifest.json":
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", "", err
			}
			manifest = string(data)
		case "node.csv":
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", "", err
			}
			nodeCSV = string(data)
		}
	}

	if manifest == "" || nodeCSV == "" {
		return "", "", errors.New("missing manifest.json or nodes.csv")
	}

	return manifest, nodeCSV, nil
}
