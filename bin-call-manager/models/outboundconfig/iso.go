package outboundconfig

import (
	"regexp"
	"strings"
)

// ISOCountryCodes is the set of valid ISO 3166 alpha-2 codes accepted in DestinationWhitelist.
// Entries must be lowercase. Add new codes here if the phonenumber library upgrade returns
// codes not yet in this map (you will be alerted by TestISOMapDrift).
var ISOCountryCodes = map[string]struct{}{
	"ac": {}, "ad": {}, "ae": {}, "af": {}, "ag": {}, "ai": {}, "al": {}, "am": {},
	"ao": {}, "aq": {}, "ar": {}, "as": {}, "at": {}, "au": {}, "aw": {}, "ax": {},
	"az": {}, "ba": {}, "bb": {}, "bd": {}, "be": {}, "bf": {}, "bg": {}, "bh": {},
	"bi": {}, "bj": {}, "bl": {}, "bm": {}, "bn": {}, "bo": {}, "bq": {}, "br": {},
	"bs": {}, "bt": {}, "bv": {}, "bw": {}, "by": {}, "bz": {}, "ca": {}, "cc": {},
	"cd": {}, "cf": {}, "cg": {}, "ch": {}, "ci": {}, "ck": {}, "cl": {}, "cm": {},
	"cn": {}, "co": {}, "cr": {}, "cu": {}, "cv": {}, "cw": {}, "cx": {}, "cy": {},
	"cz": {}, "de": {}, "dj": {}, "dk": {}, "dm": {}, "do": {}, "dz": {}, "ec": {},
	"ee": {}, "eg": {}, "eh": {}, "er": {}, "es": {}, "et": {}, "fi": {}, "fj": {},
	"fk": {}, "fm": {}, "fo": {}, "fr": {}, "ga": {}, "gb": {}, "gd": {}, "ge": {},
	"gf": {}, "gg": {}, "gh": {}, "gi": {}, "gl": {}, "gm": {}, "gn": {}, "gp": {},
	"gq": {}, "gr": {}, "gs": {}, "gt": {}, "gu": {}, "gw": {}, "gy": {}, "hk": {},
	"hm": {}, "hn": {}, "hr": {}, "ht": {}, "hu": {}, "id": {}, "ie": {}, "il": {},
	"im": {}, "in": {}, "io": {}, "iq": {}, "ir": {}, "is": {}, "it": {}, "je": {},
	"jm": {}, "jo": {}, "jp": {}, "ke": {}, "kg": {}, "kh": {}, "ki": {}, "km": {},
	"kn": {}, "kp": {}, "kr": {}, "kw": {}, "ky": {}, "kz": {}, "la": {}, "lb": {},
	"lc": {}, "li": {}, "lk": {}, "lr": {}, "ls": {}, "lt": {}, "lu": {}, "lv": {},
	"ly": {}, "ma": {}, "mc": {}, "md": {}, "me": {}, "mf": {}, "mg": {}, "mh": {},
	"mk": {}, "ml": {}, "mm": {}, "mn": {}, "mo": {}, "mp": {}, "mq": {}, "mr": {},
	"ms": {}, "mt": {}, "mu": {}, "mv": {}, "mw": {}, "mx": {}, "my": {}, "mz": {},
	"na": {}, "nc": {}, "ne": {}, "nf": {}, "ng": {}, "ni": {}, "nl": {}, "no": {},
	"np": {}, "nr": {}, "nu": {}, "nz": {}, "om": {}, "pa": {}, "pe": {}, "pf": {},
	"pg": {}, "ph": {}, "pk": {}, "pl": {}, "pm": {}, "pn": {}, "pr": {}, "ps": {},
	"pt": {}, "pw": {}, "py": {}, "qa": {}, "re": {}, "ro": {}, "rs": {}, "ru": {},
	"rw": {}, "sa": {}, "sb": {}, "sc": {}, "sd": {}, "se": {}, "sg": {}, "sh": {},
	"si": {}, "sj": {}, "sk": {}, "sl": {}, "sm": {}, "sn": {}, "so": {}, "sr": {},
	"ss": {}, "st": {}, "sv": {}, "sx": {}, "sy": {}, "sz": {}, "tc": {}, "td": {},
	"tf": {}, "tg": {}, "th": {}, "tj": {}, "tk": {}, "tl": {}, "tm": {}, "tn": {},
	"to": {}, "tr": {}, "tt": {}, "tv": {}, "tw": {}, "tz": {}, "ua": {}, "ug": {},
	"um": {}, "us": {}, "uy": {}, "uz": {}, "va": {}, "vc": {}, "ve": {}, "vg": {},
	"vi": {}, "vn": {}, "vu": {}, "wf": {}, "ws": {}, "xk": {}, "ye": {}, "yt": {},
	"za": {}, "zm": {}, "zw": {},
}

// codecsRegexp validates the codecs field format: alphanumeric tokens (optionally
// containing hyphens) separated by commas. Hyphens are permitted because real-world
// SDP/RTP codec names use them — for example AMR-WB, GSM-EFR, telephone-event.
// A leading or trailing hyphen on a token is rejected by the surrounding `+`
// quantifiers, so each token must start and end with an alphanumeric character.
var codecsRegexp = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*(,[A-Za-z0-9]+(-[A-Za-z0-9]+)*)*$`)

// ValidateCodecs returns true if the codecs string is empty (server default) or
// matches the expected comma-separated codec token format.
func ValidateCodecs(codecs string) bool {
	if codecs == "" {
		return true
	}
	if len(codecs) > 255 {
		return false
	}
	if strings.ContainsAny(codecs, "\r\n") {
		return false
	}
	return codecsRegexp.MatchString(codecs)
}
