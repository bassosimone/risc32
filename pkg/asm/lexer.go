package asm

import (
	"bufio"
	"io"
	"regexp"
)

// LexerRule is a rule for lexing RiSC-32 assembly code.
type LexerRule struct {
	Emit bool           // emit the token?
	RE   *regexp.Regexp // regexp to test
	Type string         // token type
}

// The following constants enumerate all token types.
const (
	LexerBlank        = "Blank"
	LexerComment      = "Comment"
	LexerEOF          = ""
	LexerEOL          = "EOL"
	LexerError        = "Error"
	LexerInvalid      = "Invalid"
	LexerLabel        = "Label"
	LexerNameOrNumber = "NameOrNumber"
)

// LexerRules contains the lexer rules. Note that all lexer rules start
// with the `^` anchor because we remove already lexed input.
var LexerRules = []LexerRule{{
	RE:   regexp.MustCompile(`^#[^\n]*`),
	Type: LexerComment,
}, {
	Emit: true,
	RE:   regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*:`),
	Type: LexerLabel,
}, {
	Emit: true,
	RE:   regexp.MustCompile(`^[.a-zA-Z_][a-zA-Z0-9_]*`),
	Type: LexerNameOrNumber,
}, {
	Emit: true,
	RE:   regexp.MustCompile(`^(0|-?[1-9][0-9]*)`),
	Type: LexerNameOrNumber,
}, {
	RE:   regexp.MustCompile(`^[ \t]+`),
	Type: LexerBlank,
}}

// LexerToken is a token found by the lexer.
type LexerToken struct {
	Err    error
	Lineno int
	Type   string
	Value  string
}

// StartLexing starts the lexer in a background goroutine.
func StartLexing(r io.Reader) <-chan LexerToken {
	output := make(chan LexerToken)
	go LexAsync(r, output)
	return output
}

// LexAsync runs the lexer and emits tokens on the out channel.
func LexAsync(r io.Reader, out chan<- LexerToken) {
	defer close(out)
	scanner := bufio.NewScanner(r)
	var lineno int
	for scanner.Scan() {
		lineno++
		LexLine(scanner.Text(), lineno, out)
	}
	if err := scanner.Err(); err != nil {
		out <- LexerToken{Lineno: lineno, Err: err}
	}
	return
}

// LexLine lexes a single line and emits tokens on the out channel.
func LexLine(text string, lineno int, out chan<- LexerToken) {
restart:
	for text != "" {
		for _, rule := range LexerRules {
			if m := rule.RE.FindStringIndex(text); m != nil {
				// Note: all rules use the ^ anchor so we are always
				// matching at the beginning of `text`.
				if rule.Emit {
					out <- LexerToken{
						Lineno: lineno,
						Type:   rule.Type,
						Value:  text[m[0]:m[1]],
					}
				}
				text = text[m[1]:]
				goto restart
			}
		}
		// If we cannot make a sense of the remainder of the line
		// just call all the remainder of the line invalid.
		out <- LexerToken{Lineno: lineno, Type: LexerInvalid}
		// But remember to insert the information about the EOL.
		break
	}
	out <- LexerToken{Lineno: lineno, Type: LexerEOL}
	return
}
