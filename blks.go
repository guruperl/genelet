package genelet

import (
	"net/url"
	"os"
	"strconv"

	"html/template"
)

func (self *Config) Sendmail(lists []map[string]interface{}, ARGS url.Values, other map[string]interface{}) error {
	for _, name := range []string{"_gmail", "_gapple", "_ggoogle", "_gsms"} {
		glo := self.Blks[name]
		loc := other[name]
		if glo == nil || loc == nil {
			continue
		}
		outmail := ""
		envelope := loc.(map[string]interface{})
		if envelope["content"] != nil {
			outmail = envelope["content"].(string)
		} else if envelope["file"] != nil {
			var tmpl *Tmpl
			if envelope["extra"] == nil {
				tmpl = &Tmpl{lists, ARGS, other, nil, true}
			} else {
				tmpl = &Tmpl{lists, ARGS, other, envelope["extra"].(map[string]interface{}), true}
			}
			T0, err := template.ParseFiles(envelope["file"].(string))
			if err != nil {
				return err
			}
			if outmail, err = tmpl.Get_page(T0); err != nil {
				return err
			}
		}
		if outmail == "" {
			return Err(1061)
		}

		headers := make(map[string]string)
		for key, val := range envelope {
			if Grep([]string{"file", "content", "callback", "extra"}, key) {
				continue
			}
			headers[key] = val.(string)
		}

		switch name {
		case "_gmail":
			smtp := new(Smtpssl)
			for k, v := range glo {
				switch k {
				case "Username":
					smtp.Username = v
				case "Password":
					smtp.Password = v
				case "Address":
					smtp.Address = v
				case "From":
					smtp.From = v
				case "InsecureSkipVerify", "TLSInsecureSkipVerify":
					enabled, err := strconv.ParseBool(v)
					if err != nil {
						return err
					}
					smtp.InsecureSkipVerify = enabled
				default:
					headers[k] = v
				}
			}
			if smtp.Username == "" {
				smtp.Username = os.Getenv("SMTPUSER")
			}
			if smtp.Password == "" {
				smtp.Password = os.Getenv("SMTPPASS")
			}
			if smtp.Address == "" {
				smtp.Address = os.Getenv("SMTPHOST")
			}
			return smtp.Send(headers, outmail)
		default:
		}
	}
	return nil
}
