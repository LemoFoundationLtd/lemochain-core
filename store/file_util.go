package store

import "os"

func IsExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func CreateFile(path string) error {
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	} else {
		return nil
	}
}
