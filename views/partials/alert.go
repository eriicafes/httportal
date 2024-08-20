package partials

import "github.com/eriicafes/tmpl"

func Alert(message, desc string) tmpl.Template {
	return alert{message, desc, false}
}

func AlertError(message, desc string) tmpl.Template {
	return alert{message, desc, true}
}

type alert struct {
	Message string
	Desc    string
	Error   bool
}

func (t alert) Template() (string, any) {
	return "partials/alert", t
}
