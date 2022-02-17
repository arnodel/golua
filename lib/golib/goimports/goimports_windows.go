//go:build windows
// +build windows

package goimports

import "errors"

const Supported = false

func LoadGoPackage(pkg string, pluginRoot string, forceBuild bool) (map[string]interface{}, error) {
	return nil, errors.New("loading a Go package not supported on Windows")
}
