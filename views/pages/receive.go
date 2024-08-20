package pages

import "github.com/eriicafes/tmpl"

type ReceiveForm struct {
	ID string
}

func (t ReceiveForm) AssociatedTemplate() (string, string, any) {
	if t.ID == "" {
		return "pages/receive", "receive-request-form", t
	}
	return "pages/receive", "receive-download-form", t
}

type ReceivePage struct {
	ID string
}

func (t ReceivePage) Template() (string, any) {
	return tmpl.Tmpl("pages/receive", RootLayout{"Receive"}, t).Template()
}
