package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/eriicafes/httportal/app"
	"github.com/eriicafes/httportal/vite"
	"github.com/eriicafes/tmpl"
)

func main() {
	vite, err := vite.New("dist", "public", "static", "5173", false)
	if err != nil {
		panic(err)
	}
	tp := tmpl.New("views").
		OnLoad(func(name string, t *template.Template) {
			t.Funcs(vite.Funcs())
		}).
		Autoload("components", "partials").
		LoadWithLayouts("pages").
		MustParse()
	app := app.New(tp, app.NewPortal())

	app.Mount(http.DefaultServeMux)
	http.Handle("GET /static/", http.StripPrefix("/static", vite.FileServer()))

	log.Println("server listening on port 8080...")
	if err = http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
