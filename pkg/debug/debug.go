package debug

import (
	"os"
	"path/filepath"
)

func DoIt(origImg, component string) (string, error) {
	sourcedir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(sourcedir)

	//pull img
	err = PullArtifact(origImg, sourcedir)
	if err != nil {
		return "", err
	}

	// update spin.toml
	//TODO(rajatjindal): what if spin.toml is not in root of oci artifact?
	updated, err := update(filepath.Join(sourcedir, "spin.toml"), component)
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())

	//write new spin.toml
	err = os.WriteFile(file.Name(), []byte(updated), 0644)
	if err != nil {
		return "", err
	}

	//copy wasm console.wasm
	//create new img
	//push img
	img, err := CreateDebugImage(origImg, file.Name())
	if err != nil {
		return "", err
	}

	return img, nil
}
