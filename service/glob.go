package service

import (
	"os"
	"path/filepath"
	"regexp"

	multierror "github.com/hashicorp/go-multierror"
)

// Glob get match files
func Glob(path, reg string) (matches []string, err error) {
	r, err := regexp.Compile(reg)
	if err != nil {
		return
	}
	matches = make([]string, 0)
	var result *multierror.Error
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			result = multierror.Append(result, err)
		}
		if info.IsDir() {
			return nil
		}
		if r.MatchString(file) {
			matches = append(matches, file)
		}
		return nil
	})
	err = result.ErrorOrNil()
	return
}
