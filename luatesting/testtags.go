package luatesting

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
)

var tagsPtn = regexp.MustCompile(`^-- tags: *(!?[a-z]+)(?:,(!?[a-z]+))* *\n`)

type testTag struct {
	name  string
	value bool
}

func checkTags(source []byte) (bool, error) {
	tags, err := getTags(source)
	if err != nil {
		return false, err
	}
	for _, tag := range tags {
		if !checkTag(tag) {
			return false, nil
		}
	}
	return true, nil
}

func getTags(source []byte) ([]testTag, error) {
	if !bytes.HasPrefix(source, []byte("-- tags:")) {
		return nil, nil
	}
	match := tagsPtn.FindSubmatch(source)
	if len(match) == 0 {
		return nil, fmt.Errorf("Bad tags line")
	}
	var tags []testTag
	for _, b := range match[1:] {
		if len(b) == 0 {
			continue
		}
		if b[0] == '!' {
			tags = append(tags, testTag{name: string(b[1:]), value: false})
		} else {
			tags = append(tags, testTag{name: string(b), value: true})
		}
	}
	return tags, nil
}

func checkTag(tag testTag) bool {
	if tag.name == "unix" {
		return unixOS[runtime.GOOS] == tag.value
	}
	if knownOS[tag.name] {
		return (tag.name == runtime.GOOS) == tag.value
	}
	return true
}

// Maps below copied from package build
var unixOS = map[string]bool{
	"aix":       true,
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"hurd":      true,
	"illumos":   true,
	"ios":       true,
	"linux":     true,
	"netbsd":    true,
	"openbsd":   true,
	"solaris":   true,
}

var knownOS = map[string]bool{
	"aix":       true,
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"hurd":      true,
	"illumos":   true,
	"ios":       true,
	"js":        true,
	"linux":     true,
	"nacl":      true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"windows":   true,
	"zos":       true,
}
