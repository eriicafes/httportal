package pages

import "github.com/eriicafes/tmpl"

type SendForm struct {
	ID string
}

func (t SendForm) AssociatedTemplate() (string, string, any) {
	if t.ID == "" {
		return "pages/send", "send-request-form", t
	}
	return "pages/send", "send-upload-form", t
}

type SendCompleted struct {
	ID string
}

func (t SendCompleted) AssociatedTemplate() (string, string, any) {
	return "pages/send", "send-completed", t
}

type SendPage struct{}

func (t SendPage) Template() (string, any) {
	return tmpl.Tmpl("pages/send", RootLayout{"Send"}, t).Template()
}
