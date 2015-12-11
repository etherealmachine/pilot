package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func buildStatic() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	static := make(map[string]string)
	if err := filepath.Walk("static", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		bs, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		static[filepath.Clean(strings.TrimPrefix(path, wd))] = string(bs)
		return nil
	}); err != nil {
		return err
	}
	f, err := os.Create("static.go")
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString("package main\nvar static = map[string]string{")
	for path, contents := range static {
		f.WriteString(fmt.Sprintf("%q", path))
		f.WriteString(": ")
		f.WriteString(fmt.Sprintf("%q", contents))
		f.WriteString(",\n")
	}
	f.WriteString("}\n")
	return nil
}
