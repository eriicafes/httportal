package pages

import "github.com/eriicafes/tmpl"

type IndexPage struct{}

func (t IndexPage) Template() (string, any) {
	return tmpl.Tmpl("pages/index", RootLayout{"Home"}, t).Template()
}
