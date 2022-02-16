package util

import (
	"github.com/mitchellh/go-homedir"
	"strings"
)

func IsCosPath(path string) bool {
	if len(path) <= 6 {
		return false
	}
	if path[:6] == "cos://" {
		return true
	} else {
		return false
	}
}

func ParsePath(url string) (bucketName string, path string) {
	if IsCosPath(url) {
		res := strings.SplitN(url[6:], "/", 2)
		if len(res) < 2 {
			return res[0], ""
		} else {
			return res[0], res[1]
		}
	} else {
		path, _ = homedir.Expand(url)
		return "", path
	}
}
