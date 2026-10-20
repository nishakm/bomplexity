package main

import (
	"errors"
	"fmt"
	"net/url"
	"runtime/debug"
	"strings"
)

// renderSBOM reads the running binary's build info and returns it as an ASCII
// tree of package URLs.
func renderSBOM() (string, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("build info unavailable (binary not built with module support)")
	}
	return formatSBOM(bi), nil
}

// formatSBOM draws build info as a tree: the main module at the root, then
// every dependency module and the Go toolchain.
//
// Build info records which modules are in the binary but not which module
// imports which, so every entry hangs directly off the main module.
func formatSBOM(bi *debug.BuildInfo) string {
	var b strings.Builder
	mainVersion := bi.Main.Version
	if mainVersion == "(devel)" {
		mainVersion = ""
	}
	b.WriteString(purl(bi.Main.Path, mainVersion) + "\n")

	purls := make([]string, 0, len(bi.Deps)+1)
	for _, d := range bi.Deps {
		m := d
		if d.Replace != nil {
			m = d.Replace
		}
		purls = append(purls, purl(d.Path, m.Version))
	}
	purls = append(purls, "pkg:golang/std@"+strings.TrimPrefix(bi.GoVersion, "go"))

	for i, p := range purls {
		branch := "+-- "
		if i == len(purls)-1 {
			branch = "`-- "
		}
		b.WriteString(branch + p + "\n")
	}
	return b.String()
}

func purl(path, version string) string {
	u := (&url.URL{Path: path}).EscapedPath()
	if version == "" {
		return "pkg:golang/" + u
	}
	return fmt.Sprintf("pkg:golang/%s@%s", u, url.PathEscape(version))
}
