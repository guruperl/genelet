package genelet

type Chartag struct {
	ContentType string
	Short       string
	Case        int8
	Challenge   string
	Logged      string
	Logout      string
	Failed      string
}

func (self Chartag) CallChallenge() string { return self.charcasestring(self.Challenge) }
func (self Chartag) CallLogged() string    { return self.charcasestring(self.Logged) }
func (self Chartag) CallLogout() string    { return self.charcasestring(self.Logout) }
func (self Chartag) CallFailed() string    { return self.charcasestring(self.Failed) }

func (self Chartag) charcasestring(in string) string {
	if self.Case == 2 {
		return `<?xml version="1.0" encoding="UTF-8"?><data>` + in + `</data>`
	} else if self.Case == 1 {
		return `{"data":"` + in + `"}`
	}
	return ""
}
