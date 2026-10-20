package main

import (
	"fmt"
	"net/url"
	"runtime/debug"
	"strings"
)

// renderSBOM turns Go build info into an ASCII tree of package URLs: the main
// module at the root, then every dependency module and the Go toolchain.
//
// Build info records which modules are in the binary but not which module
// imports which, so every entry hangs directly off the main module.
func renderSBOM(bi *debug.BuildInfo) string {
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
