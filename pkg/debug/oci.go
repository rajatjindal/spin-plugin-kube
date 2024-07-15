package debug

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func CreateDebugImage(sourceRef, spinTomlFile string) (string, error) {
	debugImg := fmt.Sprintf("ttl.sh/rajatjindal/wasm-console-debug-%d:24h", time.Now().Unix())
	ociPathDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	fmt.Println(ociPathDir)
	// defer os.RemoveAll(ociPathDir.Name())

	path, err := layout.Write(filepath.Join(".", ociPathDir), emptyIndex())
	if err != nil {
		return "", fmt.Errorf("failed to create OCI layout: %v", err)
	}

	ref, err := name.ParseReference(sourceRef)
	if err != nil {
		return "", fmt.Errorf("failed to parse base image reference: %v", err)
	}

	img, err := remote.Image(ref)
	if err != nil {
		return "", fmt.Errorf("failed to pull base image: %v", err)
	}

	newLayer, err := createLayerWithNewFile(spinTomlFile, "spin.toml")
	if err != nil {
		return "", fmt.Errorf("failed to create new layer: %v", err)
	}

	newimg, err := mutate.Append(img, mutate.Addendum{
		Layer: newLayer,
		History: v1.History{
			Author:    "rajatjindal83@gmail.com",
			CreatedBy: "add updated spin.toml",
			Created:   v1.Time{Time: time.Now()},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to append new layer: %v", err)
	}

	if err := path.AppendImage(newimg); err != nil {
		return "", fmt.Errorf("failed to append image to OCI layout: %v", err)
	}

	destRef, err := name.ParseReference(debugImg)
	if err != nil {
		return "", fmt.Errorf("failed to parse destination reference: %v", err)
	}

	options := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}

	if err := remote.Write(destRef, newimg, options...); err != nil {
		return "", fmt.Errorf("failed to push image: %v", err)
	}

	return debugImg, nil
}

func emptyIndex() v1.ImageIndex {
	return empty.Index
}

func PullArtifact(source, localdir string) error {
	imageName := source
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return fmt.Errorf("failed to parse reference: %v", err)
	}

	auth := authn.DefaultKeychain
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(auth))
	if err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("failed to find layers: %v", err)
	}
	for _, l := range layers {
		reader, _ := l.Uncompressed()
		err = ExtractTarGzTo(reader, localdir)
		if err != nil {
			return fmt.Errorf("failed to extract layer: %v", err)
		}
	}

	return nil
}

func ExtractTarGzTo(uncompressedStream io.Reader, todir string) error {
	tarReader := tar.NewReader(uncompressedStream)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path.Join(todir, header.Name), 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			outFile, err := os.Create(path.Join(todir, header.Name))
			if err != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %s", err.Error())
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("ExtractTarGz: Copy() failed: %s", err.Error())
			}
		default:
			return fmt.Errorf(
				"ExtractTarGz: uknown type: %v in %s",
				header.Typeflag,
				header.Name)
		}
	}

	return nil
}

func build(filename, atPath string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		pw.CloseWithError(func() error {
			w := tar.NewWriter(pw)

			for _, entry := range []string{filename} {
				if err := func(entry string) error {
					in, err := os.Open(entry)
					if err != nil {
						return err
					}

					defer in.Close()

					st, err := in.Stat()
					if err != nil {
						return err
					}

					if err = w.WriteHeader(&tar.Header{
						Name: atPath,
						Size: st.Size(),
						Mode: 0644,
					}); err != nil {
						return err
					}

					_, err = io.Copy(w, in)
					if err != nil {
						return err
					}

					return in.Close()
				}(entry); err != nil {
					return err
				}
			}

			return w.Close()
		}())
	}()

	return pr
}

func createLayerWithNewFile(filename, atPath string) (v1.Layer, error) {
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return build(filename, atPath), nil
	})
}
