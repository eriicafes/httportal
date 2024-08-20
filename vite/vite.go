package vite

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ManifestChunk struct {
	Src            string
	File           string
	Css            []string
	Assets         []string
	IsEntry        bool
	Name           string
	IsDynamicEntry bool
	Imports        []string
	DynamicImports []string
}

type Vite struct {
	Manifest   map[string]ManifestChunk
	staticPath string
	dev        bool
	port       string
	publicFS   fs.FS
	outputFS   fs.FS
}

// New creates a new vite instance.
//
// output is the directory relative from root where build output will be placed.
// Should match viteConfig.build.outDir which has a default of "dist".
//
// public is the directory to serve as plain static assets.
// Files in this directory are served and copied to build dist dir as-is without transform.
// Should match viteConfig.publicDir which has a default of "public".
//
// staticPath is the path your application will be serving static assets.
//
// dev indicates if vite is running in developement mode. If set to false production vite tags will be placed in the html head.
// Run `vite build` and set dev to false for production.
//
// The minimum requirement for vite to work is to render {{ vite "input" }} in the html head preferrably in a layout
// (where input is the path to vite entry point as configured in viteConfig.build.rollupOptions.input).
// Multiple entry points can be specified using {{ vite "input1" "input2" }}.
func New(output string, public string, staticPath string, port string, dev bool) (Vite, error) {
	return NewFS(os.DirFS(output), os.DirFS(public), staticPath, port, dev)
}

// NewFS creates a new vite instance.
//
// output is the directory relative from root where build output will be placed.
// Should match viteConfig.build.outDir which has a default of "dist".
//
// public is the directory to serve as plain static assets.
// Files in this directory are served and copied to build dist dir as-is without transform.
// Should match viteConfig.publicDir which has a default of "public".
//
// staticPath is the path your application will be serving static assets.
//
// dev indicates if vite is running in developement mode. If set to false production vite tags will be placed in the html head.
// Run `vite build` and set dev to false for production.
//
// The minimum requirement for vite to work is to render {{ vite "input" }} in the html head preferrably in a layout
// (where input is the path to vite entry point as configured in viteConfig.build.rollupOptions.input).
// Multiple entry points can be specified using {{ vite "input1" "input2" }}.
func NewFS(output fs.FS, public fs.FS, staticPath string, port string, dev bool) (Vite, error) {
	m := make(map[string]ManifestChunk)
	var err error
	if !dev {
		b, rerr := fs.ReadFile(output, ".vite/manifest.json")
		err = rerr
		if err == nil {
			err = json.Unmarshal(b, &m)
		}
	}
	return Vite{
		Manifest:   m,
		staticPath: filepath.Join("/", staticPath),
		dev:        dev,
		port:       port,
		outputFS:   output,
		publicFS:   public,
	}, err
}

// Funcs returns vite helper functions for templates.
//
// vite returns required vite tags to be rendered in the html head.
// Usage: {{ vite "input" }} or {{ vite "input1" "input2" }} for multiple entry points.
//
// public returns the absolute path for an asset in the public directory.
// Usage: {{ public "logo.png" }}.
//
// assets returns the absolute path for an asset in vite entry point viteConfig.build.rollupOptions.input.
// Use for assets that are not already required when rendering vite tags.
// Usage: {{ assets "src/main.ts" }}.
func (v *Vite) Funcs() template.FuncMap {
	return template.FuncMap{
		"vite":   v.RenderViteTags,
		"public": v.PublicPath,
		"assets": v.AssetPath,
	}
}

// PublicPath returns the absolute path for an asset in the public directory.
func (v *Vite) PublicPath(path string) string {
	return filepath.Join(v.staticPath, strings.TrimSpace(path))
}

// AssetPath returns the absolute path for an asset in vite entry point viteConfig.build.rollupOptions.input.
// During development AssetPath returns the file name as is.
func (v *Vite) AssetPath(name string) (string, error) {
	if v.dev {
		return filepath.Join("/", strings.TrimSpace(name)), nil
	}
	chunk, ok := v.Manifest[name]
	if !ok {
		return "", fmt.Errorf("asset %q does not exist in vite manifest", name)
	}
	return filepath.Join(v.staticPath, chunk.File), nil
}

func appendTag(tags *strings.Builder, s string) (int, error) {
	if tags.Len() > 0 {
		tags.WriteString("\n\t")
	}
	return tags.WriteString(s)
}

// RenderViteTags returns required vite tags to be rendered in the html head.
func (v *Vite) RenderViteTags(inputs ...string) (template.HTML, error) {
	var tags strings.Builder

	if v.dev {
		appendTag(&tags, fmt.Sprintf("<script type=\"module\" src=\"http://localhost:%s/@vite/client\"></script>", v.port))
		for _, input := range inputs {
			path, err := v.AssetPath(input)
			if err != nil {
				return "", err
			}
			appendTag(&tags, fmt.Sprintf("<script type=\"module\" src=\"http://localhost:%s%s\"></script>", v.port, path))
		}
		return template.HTML(tags.String()), nil
	}

	for _, input := range inputs {
		chunk, ok := v.Manifest[input]
		if !ok || !chunk.IsEntry {
			return "", fmt.Errorf("entry point %q does not exist in vite manifest", input)
		}

		for _, css := range chunk.Css {
			appendTag(&tags, fmt.Sprintf("<link rel=\"stylesheet\" href=\"%s\" />", v.PublicPath(css)))
		}
		chunks := v.importedChunks(&chunk)
		for _, ch := range chunks {
			for _, css := range ch.Css {
				appendTag(&tags, fmt.Sprintf("<link rel=\"stylesheet\" href=\"%s\" />", v.PublicPath(css)))
			}
		}
		appendTag(&tags, fmt.Sprintf("<script type=\"module\" src=\"%s\"></script>", v.PublicPath(chunk.File)))
		for _, ch := range chunks {
			appendTag(&tags, fmt.Sprintf("<link rel=\"modulepreload\" href=\"%s\" />", v.PublicPath(ch.File)))
		}
	}
	return template.HTML(tags.String()), nil
}

func (v *Vite) importedChunks(chunk *ManifestChunk) []*ManifestChunk {
	var chunks []*ManifestChunk
	for _, name := range chunk.DynamicImports {
		if ch, ok := v.Manifest[name]; ok {
			chunks = append(chunks, &ch)
		}
	}
	for _, name := range chunk.Imports {
		if ch, ok := v.Manifest[name]; ok {
			chunks = append(chunks, &ch)
		}
	}
	return chunks
}

// FileServer will serve vite static assets.
//
// In development vite will forward requests to static assets to your application, FileServer will serve the public directory.
//
// In production running `vite build` will copy public assets to the dist directory as well as other assets, FileServer will serve the dist directory.
func (v *Vite) FileServer() http.Handler {
	if v.dev {
		return ignoreDirListingMiddleware(http.FileServerFS(v.publicFS))
	} else {
		return ignoreDirListingMiddleware(http.FileServerFS(v.outputFS))
	}
}

func ignoreDirListingMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
