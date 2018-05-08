package packagelib

import (
	"os"
	"strings"

	rt "github.com/arnodel/golua/runtime"
)

func searchpath(name, path string, sep, rep string) (string, []string) {
	namePath := strings.Replace(name, sep, rep, -1)
	templates := strings.Split(path, ";")
	for i, template := range templates {
		searchpath := strings.Replace(template, "?", namePath, -1)
		f, err := os.Open(searchpath)
		f.Close()
		if err == nil {
			return searchpath, nil
		}
		templates[i] = searchpath
	}
	return "", templates
}

func searchPreload(t *rt.Thread, c *rt.GoCont) (rt.Cont, *rt.Error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err.AddContext(c)
	}
	s, err := c.StringArg(0)
	if err != nil {
		return nil, err.AddContext(c)
	}
	_ = s
	return nil, nil
}
