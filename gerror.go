package genelet

import (
	"fmt"
	"net/http"
)

type Gerror struct {
	Code   int
	Errstr string
}

func Err(code int, str ...string) Gerror {
	if str != nil {
		return Gerror{code, str[0]}
	}
	return Gerror{code, ""}
}

func (self *Gerror) Refresh() {
	if self == nil {
		return
	}
	if self.Code < 1000 {
		self.Errstr = http.StatusText(self.Code)
	} else if self.Errstr == "" {
		self.Errstr = self.Error()
	}
}

func (self Gerror) Error() string {
	errs := map[int]string{
		1000: "Application error.",
		1001: "Google authorization required.",
		1002: "Facebook authorization required.",
		1003: "User denied authorization.",
		1004: "Failed in browser getting token.",
		1005: "Failed in browser getting app.",
		1006: "Failed in browser refreshing token.",
		1007: "Failed in browser refreshing app.",
		1008: "Failed in finding token.",
		1009: "Twitter athorization required.",
		1010: "Failed in retrieve token secret from db for twitter.",
		1011: "Failed in getting user_id from twitter.",
		1013: "Failed to get ticket from box.",

		1020: "Login required.",
		1021: "Not authorized to view the page.",
		1022: "Login is expired.",
		1023: "Your IP does not match the login credential.",
		1024: "Login signature is not acceptable.",
		1025: "Sign In to your account",

		1030: "Too many failed logins.",
		1031: "Login incorrect. Please try again.",
		1032: "System error.",
		1033: "Web server configuration error.",
		1034: "Login failed. Please try again.",
		1035: "This input field is missing: ",
		1036: "Please make sure your browser supports cookie.",
		1037: "Missing input.",

		1040: "Empty field.",
		1041: "Foreign key forced but its value not provided.",
		1042: "Foreign key fields and foreign key-to-be fields do not match.",
		1043: "Variable undefined in your customzied method.",
		1044: "Variable undefined in your procedure method.",
		1045: "Upload field not found.",

		1051: "Object method does not exist.",
		1052: "Foreign key is broken.",
		1053: "Foreign key session expired.",
		1054: "Signature field not found.",
		1055: "Signature not found.",
		1056: "Signature column not found.",

		1060: "Email Server, Sender, From, To and Subject must be existing.",
		1061: "Message is empty.",
		1062: "Sending mail failed.",
		1063: "Mail server not reachable.",
		1064: "No message nor template.",

		1070: "Multiple records found in insupd.",
		1071: "Select Syntax error.",
		1072: "Failed to connect to the database.",
		1073: "SQL failed, check your SQL statement; or duplicate entry.",

		1171: "Insert failed, the column may exist",
		1172: "Delete failed, maybe foreign keyed",
		1173: "Update failed",
		1174: "Select condition failed",
		1175: "PROCEDURE format incorrect",

		1074: "Die from db.",
		1075: "Records exist in other tables",
		1076: "Could not get a random ID.",
		1077: "Condition not found in update.",
		1078: "Hash not found in insert.",
		1079: "Missing lists.",
		1080: "Can't write to cache.",

		1090: "No socket.",
		1091: "Can't connect to socket.",
		1092: "SSL error.",

		1100: "Sender signature not found.",
		1101: "Sender signature not confirmed.",
		1102: "Invalid JSON.",
		1103: "Incompatible JSON.",
		1105: "Not allowed to send.",
		1106: "Inactive recipient.",
		1107: "Bounce not found.",
		1108: "Bounce query exception.",
		1109: "JSON required.",
		1110: "Too many batch messages.",
		1111: "HTTP email server error.",
		1113: "Invalid email request.",
	}
	if self.Code != 0 && self.Errstr != "" {
		return self.Errstr //fmt.Sprintf("%d: %s", self.Code, self.Errstr)
	} else if self.Code != 0 && errs[self.Code] != "" {
		return errs[self.Code] //fmt.Sprintf("%d: %s", self.Code, errs[self.Code])
	} else if self.Code != 0 {
		return fmt.Sprintf("%d", self.Code)
	} else if self.Errstr != "" {
		return self.Errstr
	}
	return ""
}
