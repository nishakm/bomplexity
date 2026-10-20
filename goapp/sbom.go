package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

// Minimal CycloneDX 1.6 JSON model: just the fields we emit.

type bom struct {
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber"`
	Version      int         `json:"version"`
	Metadata     bomMetadata `json:"metadata"`
	Components   []component `json:"components"`
}

type bomMetadata struct {
	Timestamp string     `json:"timestamp"`
	Tools     bomTools   `json:"tools"`
	Component component  `json:"component"`
	Props     []property `json:"properties,omitempty"`
}

type bomTools struct {
	Components []component `json:"components"`
}

type component struct {
	Type       string     `json:"type"`
	BOMRef     string     `json:"bom-ref,omitempty"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	PURL       string     `json:"purl,omitempty"`
	Hashes     []hash     `json:"hashes,omitempty"`
	Properties []property `json:"properties,omitempty"`
}

type hash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// buildSBOM turns Go build info into a flat CycloneDX document: the main
// module, every dependency module (version + checksum) and the Go toolchain.
// It carries no dependency graph: build info does not record who imports whom.
func buildSBOM(bi *debug.BuildInfo) bom {
	main := moduleComponent(bi.Main.Path, bi.Main.Version, bi.Main.Sum, "application")
	if main.Version == "" || main.Version == "(devel)" {
		main.Version = "(devel)"
		main.PURL = purl(bi.Main.Path, "")
	}

	comps := []component{}
	for _, d := range bi.Deps {
		m := d
		if d.Replace != nil {
			m = d.Replace
		}
		c := moduleComponent(d.Path, m.Version, m.Sum, "library")
		if d.Replace != nil {
			c.Properties = append(c.Properties, property{"go:replaced-by", m.Path + "@" + m.Version})
		}
		comps = append(comps, c)
	}

	goVer := strings.TrimPrefix(bi.GoVersion, "go")
	comps = append(comps, component{
		Type:    "platform",
		BOMRef:  "go-toolchain",
		Name:    "go",
		Version: goVer,
		PURL:    "pkg:golang/std@" + goVer,
	})

	return bom{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.6",
		SerialNumber: "urn:uuid:" + newUUID(),
		Version:      1,
		Metadata: bomMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: bomTools{Components: []component{{
				Type: "application", Name: "guestbook /sbom endpoint",
				Properties: []property{{"source", "runtime/debug.ReadBuildInfo"}},
			}}},
			Component: main,
		},
		Components: comps,
	}
}

func moduleComponent(path, version, sum, typ string) component {
	c := component{
		Type:    typ,
		BOMRef:  purl(path, version),
		Name:    path,
		Version: version,
		PURL:    purl(path, version),
	}
	// go.sum "h1:" values are base64 of a SHA-256 over the module's file
	// hashes. CycloneDX wants hex, so convert and keep the original too.
	if raw, ok := strings.CutPrefix(sum, "h1:"); ok {
		if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
			c.Hashes = []hash{{Alg: "SHA-256", Content: hex.EncodeToString(b)}}
			c.Properties = append(c.Properties, property{"go:sum", sum})
		}
	}
	return c
}

func purl(path, version string) string {
	u := (&url.URL{Path: path}).EscapedPath()
	if version == "" {
		return "pkg:golang/" + u
	}
	return fmt.Sprintf("pkg:golang/%s@%s", u, url.PathEscape(version))
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
