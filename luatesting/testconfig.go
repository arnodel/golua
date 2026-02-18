package luatesting

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

var configPtn = regexp.MustCompile(`^-- config: *([a-z]+(?:,[a-z]+)*) *[\n\r]`)

var configOptions = map[string]rt.RuntimeOption{
	"reservedglobal": rt.WithReservedGlobal(),
}

func extractConfig(source []byte) ([]rt.RuntimeOption, error) {
	if !bytes.HasPrefix(source, []byte("-- config:")) {
		return nil, nil
	}
	match := configPtn.FindSubmatch(source)
	if len(match) == 0 {
		return nil, fmt.Errorf("bad config line")
	}
	names := strings.Split(string(match[1]), ",")
	var opts []rt.RuntimeOption
	for _, name := range names {
		opt, ok := configOptions[name]
		if !ok {
			return nil, fmt.Errorf("unknown config option: %s", name)
		}
		opts = append(opts, opt)
	}
	return opts, nil
}
