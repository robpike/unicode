// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Unicode is a command-line tool for studying Unicode characters.

usage: unicode [-c] [-d] [-n] [-t]

	-c: args are hex; output characters (xyz)
	-n: args are characters; output hex (23 or 23-44)
	-g: args are regular expressions for matching names
	-d: output textual description
	-t: output plain text, not one char per line
	-U: output full Unicode description

Default behavior sniffs the arguments to select -c vs. -n.
*/
package main // import "robpike.io/cmd/unicode"

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	doNum  = flag.Bool("n", false, "output numeric values")
	doChar = flag.Bool("c", false, "output characters")
	doText = flag.Bool("t", false, "output plain text")
	doDesc = flag.Bool("d", false, "describe the characters from the Unicode database, in simple form")
	doUnic = flag.Bool("u", false, "describe the characters from the Unicode database, in Unicode form")
	doUNIC = flag.Bool("U", false, "describe the characters from the Unicode database, in glorious detail")
	doGrep = flag.Bool("g", false, "grep for argument string in data")
)

var printRange = false

//go:generate sh -c "curl http://ftp.unicode.org/Public/UNIDATA/UnicodeData.txt >UnicodeData.txt"
var (
	//go:embed UnicodeData.txt
	unicodeDataTxt string
	unicodeLines   = splitLines(unicodeDataTxt)
)

func main() {
	flag.Usage = usage
	flag.Parse()
	mode()
	var codes []rune
	switch {
	case *doGrep:
		codes = argsAreRegexps()
	case *doChar:
		codes = argsAreNumbers()
	case *doNum:
		codes = argsAreChars()
	}
	if *doUnic || *doUNIC || *doDesc {
		desc(codes)
		return
	}
	if *doText {
		fmt.Printf("%s\n", string(codes))
		return
	}
	b := new(bytes.Buffer)
	for i, c := range codes {
		switch {
		case printRange:
			fmt.Fprintf(b, "%.4x %c", c, c)
			if i%4 == 3 {
				fmt.Fprint(b, "\n")
			} else {
				fmt.Fprint(b, "\t")
			}
		case *doChar:
			fmt.Fprintf(b, "%c\n", c)
		case *doNum:
			fmt.Fprintf(b, "%.4x\n", c)
		}
	}
	if b.Len() > 0 && b.Bytes()[b.Len()-1] != '\n' {
		fmt.Fprint(b, "\n")
	}
	fmt.Print(b)
}

func fatalf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(2)
}

const usageText = `usage: unicode [-c] [-d] [-n] [-t]
-c: args are hex; output characters (xyz)
-n: args are characters; output hex (23 or 23-44)
-g: args are regular expressions for matching names
-d: output textual description
-t: output plain text, not one char per line
-U: output full Unicode description

Default behavior sniffs the arguments to select -c vs. -n.
`

func usage() {
	fatalf("%s", usageText)
}

// Mode determines whether we have numeric or character input.
// If there are no flags, we sniff the first argument.
func mode() {
	if len(flag.Args()) == 0 {
		usage()
	}
	// If grepping names, we need an output format defined; default is numeric.
	if *doGrep && !(*doNum || *doChar || *doDesc || *doUnic || *doUNIC) {
		*doNum = true
	}
	if *doNum || *doChar {
		return
	}
	alldigits := true
	numDash := 0
	for _, r := range strings.Join(flag.Args(), "") {
		if !strings.ContainsRune("0123456789abcdefABCDEF-", r) {
			alldigits = false
		}
		if r == '-' {
			numDash++
		}
	}
	// If there is one '-' it's a range; if zero it's just a hex number.
	if alldigits && numDash <= 1 {
		*doChar = true
		return
	}
	*doNum = true
}

func argsAreChars() []rune {
	var codes []rune
	for i, a := range flag.Args() {
		for _, r := range a {
			codes = append(codes, r)
		}
		// Add space between arguments if output is plain text.
		if *doText && i < len(flag.Args())-1 {
			codes = append(codes, ' ')
		}
	}
	return codes
}

func argsAreNames() []rune {
	var codes []rune
	for i, a := range flag.Args() {
		for _, r := range a {
			codes = append(codes, r)
		}
		// Add space between arguments if output is plain text.
		if *doText && i < len(flag.Args())-1 {
			codes = append(codes, ' ')
		}
	}
	return codes
}

func parseRune(s string) rune {
	r, err := strconv.ParseInt(s, 16, 22)
	if err != nil {
		fatalf("%s", err)
	}
	return rune(r)
}

func argsAreNumbers() []rune {
	var codes []rune
	for _, a := range flag.Args() {
		if s := strings.Split(a, "-"); len(s) == 2 {
			printRange = true
			r1 := parseRune(s[0])
			r2 := parseRune(s[1])
			if r2 < r1 {
				usage()
			}
			for ; r1 <= r2; r1++ {
				codes = append(codes, r1)
			}
			continue
		}
		codes = append(codes, parseRune(a))
	}
	return codes
}

func argsAreRegexps() []rune {
	var codes []rune
	for _, a := range flag.Args() {
		re, err := regexp.Compile(a)
		if err != nil {
			fatalf("%s", err)
		}
		for i, line := range unicodeLines {
			fields := strings.Split(strings.ToLower(line), ";")
			line = fields[0] + "\t" + fields[1]
			if fields[10] != "" {
				line += "; " + fields[10]
			}
			if re.MatchString(line) {
				r, _ := runeOfLine(i, line)
				codes = append(codes, r)
			}
		}
	}
	return codes
}

func splitLines(text string) []string {
	lines := strings.Split(text, "\n")
	// We get an empty final line; drop it.
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func runeOfLine(i int, line string) (r rune, tab int) {
	tab = strings.IndexAny(line, "\t;")
	if tab < 0 {
		fatalf("malformed database: line %d", i)
	}
	return parseRune(line[0:tab]), tab
}

func desc(codes []rune) {
	runeData := make(map[rune]string)
	for i, l := range unicodeLines {
		r, tab := runeOfLine(i, l)
		runeData[r] = l[tab+1:]
	}
	if *doUNIC {
		for _, r := range codes {
			fmt.Printf("%#U %s", r, dumpUnicode(runeData[r]))
		}
	} else if *doUnic {
		for _, r := range codes {
			fmt.Printf("%#U %s\n", r, runeData[r])
		}
	} else {
		for _, r := range codes {
			fields := strings.Split(strings.ToLower(runeData[r]), ";")
			desc := fields[0]
			if len(desc) >= 9 && fields[9] != "" {
				desc += "; " + fields[9]
			}
			fmt.Printf("%#U %s\n", r, desc)
		}
	}
}

var prop = [...]string{
	"",
	"category: ",
	"canonical combining classes: ",
	"bidirectional category: ",
	"character decomposition mapping: ",
	"decimal digit value: ",
	"digit value: ",
	"numeric value: ",
	"mirrored: ",
	"Unicode 1.0 name: ",
	"10646 comment field: ",
	"uppercase mapping: ",
	"lowercase mapping: ",
	"titlecase mapping: ",
}

func dumpUnicode(s string) []byte {
	fields := strings.Split(s, ";")
	if len(fields) == 0 {
		return []byte{'\n'}
	}
	b := new(bytes.Buffer)
	if len(fields) != len(prop) {
		fmt.Fprintf(b, "%s: can't print: expected %d fields, got %d\n", s, len(prop), len(fields))
		return b.Bytes()
	}
	for i, f := range fields {
		if f == "" {
			continue
		}
		if i > 0 {
			b.WriteByte('\t')
		}
		fmt.Fprintf(b, "%s%s\n", prop[i], f)
	}
	return b.Bytes()
}
