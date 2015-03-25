package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Configuration struct {
	m map[string]string
}

func DefaultConfiguration() *Configuration {
	conf := new(Configuration)

	conf.m = make(map[string]string)
	conf.m["database_path"] = filepath.Join(os.Getenv("HOME"), ".taggerdb")
	conf.m["root_path"] = os.Getenv("HOME")

	return conf
}

func DefaultPath() string {
	return filepath.Join(os.Getenv("HOME"), ".taggerrc")
}

func (c *Configuration) DatabasePath() string {
	return c.m["database_path"]
}

func (c *Configuration) RootPath() string {
	return c.m["root_path"]
}

func (c *Configuration) Read(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if err := c.readEntry(line); err != nil {
			return err
		}
	}

	return nil
}

func (c *Configuration) readEntry(line string) error {
	if strings.Count(line, "=") != 1 {
		return errors.New("tagger-cli: There can only be one '=' per line in configuration")
	}

	parts := strings.Split(line, "=")
	key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	c.m[key] = value

	return nil
}

func (c *Configuration) Write(w io.Writer) error {
	keys, values := c.keyValues()
	for idx := 0; idx < len(keys); idx++ {
		key, value := keys[idx], values[idx]
		if len(key) <= 0 || len(value) <= 0 {
			continue
		}

		_, err := fmt.Fprintf(w, "%s = %s\n", key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Configuration) keyValues() ([]string, []string) {
	var keys []string
	for key, _ := range c.m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, len(keys))
	for idx, key := range keys {
		value := c.m[key]
		values[idx] = value
	}

	return keys, values
}
