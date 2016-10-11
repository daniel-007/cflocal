package utils

import (
	"archive/tar"
	"bytes"
	"encoding/binary"
	"io"
)

func CopyStream(dst io.Writer, src io.Reader, prefix string) {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(src, header); err != nil {
			break
		}
		if n, err := io.WriteString(dst, prefix); err != nil || n != len(prefix) {
			break
		}
		if _, err := io.CopyN(dst, src, int64(binary.BigEndian.Uint32(header[4:]))); err != nil {
			break
		}
	}
}

func TarFile(name string, contents []byte) (io.Reader, error) {
	tarBuffer := &bytes.Buffer{}
	tarball := tar.NewWriter(tarBuffer)
	defer tarball.Close()
	header := &tar.Header{
		Name: name,
		Size: int64(len(contents)),
		Mode: 0644,
	}
	if err := tarball.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tarball.Write(contents); err != nil {
		return nil, err
	}
	return tarBuffer, nil
}

func FileFromTar(name string, archive io.Reader) (io.Reader, error) {
	tarball := tar.NewReader(archive)
	for {
		header, err := tarball.Next()
		if err != nil {
			return nil, err
		}
		if header.Name == name {
			break
		}
	}
	return tarball, nil
}
