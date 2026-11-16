package main

import (
	"strings"
	"unicode"
)

const (
	errLambName = "lamb-name"
	errBadName  = "bad-name"
)

// Moderate list of offensive slurs, severe profanity, and explicit terms
// disallowed on public leaderboards. Checked by word/token boundaries and
// explicit compound roots to avoid false positives on innocent names.
var offensiveExactWords = map[string]struct{}{
	"anal": {}, "anus": {}, "arse": {}, "arsehole": {}, "ass": {}, "asshole": {},
	"bastard": {}, "bitch": {}, "blowjob": {}, "boner": {}, "boob": {}, "boobs": {},
	"bullshit": {}, "clit": {}, "cock": {}, "cocksucker": {}, "coon": {}, "cunt": {},
	"dick": {}, "dickhead": {}, "dildo": {}, "dipshit": {}, "dumbass": {}, "dyke": {},
	"fag": {}, "faggot": {}, "fack": {}, "fck": {}, "fckr": {}, "feltch": {}, "fuck": {},
	"fucker": {}, "fucking": {}, "fuk": {}, "fuker": {}, "phuck": {}, "phuk": {}, "gangbang": {}, "goddamn": {},
	"handjob": {}, "hentai": {}, "hoe": {}, "homo": {}, "horny": {}, "incest": {},
	"jackass": {}, "jizz": {}, "kike": {}, "kkk": {}, "kys": {}, "labia": {},
	"masturbate": {}, "milf": {}, "motherfucker": {}, "mofo": {}, "nazi": {},
	"nigga": {}, "nigger": {}, "nude": {}, "nudes": {}, "orgasm": {}, "penis": {},
	"porn": {}, "porno": {}, "prick": {}, "pussy": {}, "rape": {}, "rapist": {},
	"retard": {}, "retarded": {}, "rimjob": {}, "scrotum": {}, "semen": {}, "sex": {},
	"shit": {}, "shithead": {}, "shitty": {}, "slut": {}, "spic": {}, "suckmydick": {},
	"tit": {}, "tits": {}, "titty": {}, "tranny": {}, "twat": {}, "vagina": {},
	"wank": {}, "wanker": {}, "wetback": {}, "whore": {},
}

// Strong unambiguous substrings checked against stripped alphanumeric lowercase + leetspeak.
var offensiveStrongRoots = []string{
	"asshole", "blowjob", "bullshit", "cocksuck", "dickhead", "dipshit", "dumbass",
	"faggot", "fuck", "gangbang", "handjob", "jackass", "masturbat", "motherfuck",
	"nigga", "nigger", "pussy", "rapist", "retard", "shithead", "wetback",
}

func IsLambName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "lamb")
}

func deLeet(r rune) rune {
	switch r {
	case '0':
		return 'o'
	case '1', '!':
		return 'i'
	case '3':
		return 'e'
	case '4', '@':
		return 'a'
	case '5', '$':
		return 's'
	case '7':
		return 't'
	default:
		return unicode.ToLower(r)
	}
}

func splitCamelAndTokens(s string) []string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	var prev rune
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '@' && r != '$' && r != '!' {
			flush()
			prev = 0
			continue
		}
		if cur.Len() > 0 && unicode.IsUpper(r) && unicode.IsLower(prev) {
			flush()
		}
		cur.WriteRune(deLeet(r))
		prev = r
	}
	flush()
	return tokens
}

func collapseRepeated(s string) string {
	var b strings.Builder
	var last rune
	run := 0
	for _, r := range s {
		if r == last {
			run++
			if run <= 2 {
				b.WriteRune(r)
			}
		} else {
			last = r
			run = 1
			b.WriteRune(r)
		}
	}
	return b.String()
}

func IsBadName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	tokens := splitCamelAndTokens(trimmed)
	var squashed strings.Builder
	for _, tok := range tokens {
		squashed.WriteString(tok)
		if _, bad := offensiveExactWords[tok]; bad {
			return true
		}
		if _, bad := offensiveExactWords[collapseRepeated(tok)]; bad {
			return true
		}
	}
	joined := squashed.String()
	joinedCollapsed := collapseRepeated(joined)
	for _, root := range offensiveStrongRoots {
		if strings.Contains(joined, root) || strings.Contains(joinedCollapsed, root) {
			return true
		}
	}
	return false
}

// ValidateChallengeName validates and cleans a player name for Challenge mode.
// Returns (cleanedName, errCode).
func ValidateChallengeName(raw string) (string, string) {
	name := cleanDisplay(raw, "Lamb", 16)
	if IsLambName(name) {
		return name, errLambName
	}
	if IsBadName(name) {
		return name, errBadName
	}
	return name, ""
}
