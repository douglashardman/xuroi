package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type contract struct {
	ContractVersion string `json:"contract_version"`
	Pages           map[string]struct {
		Template         string   `json:"template"`
		Fixture          string   `json:"fixture"`
		RequiredRegions  []string `json:"required_regions"`
	} `json:"pages"`
	Partials map[string]struct {
		Template       string   `json:"template"`
		RequiredFields []string `json:"required_fields"`
	} `json:"partials"`
	Tokens struct {
		File              string   `json:"file"`
		RequiredVariables []string `json:"required_variables"`
	} `json:"tokens"`
}

func main() {
	themeDir := flag.String("theme", "", "path to theme directory (e.g. themes/puttertalk)")
	contractPath := flag.String("contract", "", "path to theme-contract.json")
	flag.Parse()

	if *themeDir == "" || *contractPath == "" {
		fmt.Fprintln(os.Stderr, "usage: themevalidate --theme <dir> --contract <theme-contract.json>")
		os.Exit(2)
	}

	data, err := os.ReadFile(*contractPath)
	if err != nil {
		fail("read contract: %v", err)
	}
	var c contract
	if err := json.Unmarshal(data, &c); err != nil {
		fail("parse contract: %v", err)
	}

	var errors []string
	checkFile := func(rel string) {
		if rel == "" {
			return
		}
		p := filepath.Join(*themeDir, rel)
		if _, err := os.Stat(p); err != nil {
			errors = append(errors, fmt.Sprintf("missing file: %s", rel))
		}
	}

	for name, page := range c.Pages {
		checkFile(page.Template)
		if page.Fixture != "" {
			fixturePath := page.Fixture
			if !filepath.IsAbs(fixturePath) {
				fixturePath = filepath.Join(filepath.Dir(*contractPath), fixturePath)
			}
			if _, err := os.Stat(fixturePath); err != nil {
				errors = append(errors, fmt.Sprintf("page %s: missing fixture %s", name, page.Fixture))
			}
		}
	}
	for name, partial := range c.Partials {
		checkFile(partial.Template)
		if len(partial.RequiredFields) == 0 {
			continue
		}
		_ = name
	}

	if c.Tokens.File != "" {
		tokensPath := filepath.Join(*themeDir, c.Tokens.File)
		tokensData, err := os.ReadFile(tokensPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("missing tokens file: %s", c.Tokens.File))
		} else {
			content := string(tokensData)
			for _, v := range c.Tokens.RequiredVariables {
				if !strings.Contains(content, v) {
					errors = append(errors, fmt.Sprintf("tokens missing variable: %s", v))
				}
			}
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintln(os.Stderr, "error:", e)
		}
		os.Exit(1)
	}
	fmt.Printf("theme %s validates against contract %s (%s)\n", *themeDir, *contractPath, c.ContractVersion)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}