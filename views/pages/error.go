package pages

import "github.com/eriicafes/tmpl"

type ErrorPage struct {
	Message string
	Desc    string
}

func (t ErrorPage) Template() (string, any) {
	return tmpl.Tmpl("pages/error", RootLayout{"Something went wrong!"}, t).Template()
}
