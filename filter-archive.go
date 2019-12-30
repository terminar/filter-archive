//
// Copyright (c) 2019 Björn Kalkbrenner <terminar@cyberphoria.org>
//           (c) 2019 Gilles Chehade <gilles@poolp.org>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
//

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"log"
	"flag"
)

var version string
var tmpFile *os.File
var bufWriter *bufio.Writer

type tx struct {
	rcptTo   []string
	action   string
	response string

	archive archiveStorage
}

type session struct {
	id       string

	rdns     string
	src      string
	heloName string
	userName string
	mtaName  string

	tx       tx
}

var sessions = make(map[string]*session)

var reporters = map[string]func(*session, []string){
	"link-connect":    linkConnect,
	"link-disconnect": linkDisconnect,
	"link-greeting":   linkGreeting,
	"link-identify":   linkIdentify,
	"link-auth":       linkAuth,
	"tx-reset":        txReset,
	"tx-begin":        txBegin,
	"tx-mail":         txMail,
	"tx-rcpt":         txRcpt,
	"tx-rollback":	   txRollback,
	"tx-envelope":	   txEnvelope,
	"timeout":         sessionTimeout,
}

var filters = map[string]func(*session, []string){
	"data": data,
	"data-line": dataLine,
}

func systemLog(text string) {
	fmt.Fprintln(os.Stderr, text)
}

func sessionTimeout(s *session, params []string) {
	log.Print("Session timeout for: " + s.id)
}

func linkConnect(s *session, params []string) {
	if len(params) != 4 {
		log.Fatal("invalid input, shouldn't happen")
	}

	s.rdns = params[0]
	s.src = params[2]
}

func linkDisconnect(s *session, params []string) {
	if len(params) != 0 {
		log.Fatal("invalid input, shouldn't happen")
	}

	delete(sessions, s.id)
}

func linkGreeting(s *session, params []string) {
	if len(params) != 1 {
		log.Fatal("invalid input, shouldn't happen")
	}

	s.mtaName = params[0]
}

func linkIdentify(s *session, params []string) {
	if len(params) != 2 {
		log.Fatal("invalid input, shouldn't happen")
	}

	s.heloName = params[1]
}

func linkAuth(s *session, params []string) {
	if len(params) != 2 {
		log.Fatal("invalid input, shouldn't happen")
	}
	if params[1] != "pass" {
		return
	}

	s.userName = params[0]
}

func txReset(s *session, params []string) {
	if len(params) != 1 {
		log.Print("message-id is missing, this may happen")
	}

	if err := s.tx.archive.Close(); err != nil {
		systemLog("ERROR: " + err.Error())
	}
	s.tx = tx{}
}


func txBegin(s *session, params []string) {
	if len(params) != 1 {
		log.Fatal("invalid input, shouldn't happen")
	}

	msgid := params[0]
	
	fname := s.id + "." + msgid
	if err := s.tx.archive.Open(fname); err != nil {
		systemLog("ERROR: can't create new file in " + archiveStoragePath + ". " + err.Error())
	}

	meta := s.tx.archive.Meta
	meta("SESSIONID=" + s.id)
	meta("MSGID=" + msgid)
	meta("MTANAME=" + s.mtaName)
	meta("HELONAME=" + s.heloName)
	meta("USERNAME=" + s.userName)
	meta("RDNS=" + s.rdns)
	meta("SRC=" + s.src)

}

func txMail(s *session, params []string) {
	if len(params) != 3 {
		log.Fatal("invalid input, shouldn't happen")
	}

	if params[2] != "ok" {
		return
	}

	s.tx.archive.Meta("FROM=" + params[1])
}

func txRcpt(s *session, params []string) {
	if len(params) != 3 {
		log.Fatal("invalid input, shouldn't happen")
	}

	if params[2] != "ok" {
		return
	}

	s.tx.rcptTo = append(s.tx.rcptTo, params[1])
}

func txEnvelope(s *session, params []string) {
	if len(params) != 2 {
		log.Fatal("invalid input, shouldn't happen")
	}

	s.tx.archive.Meta("ENVELOPEID=" + params[1])
}

func txRollback(s *session, params []string) {
	if len(params) != 1 {
		log.Fatal("invalid input, shouldn't happen")
	}

	s.tx.archive.Meta("STATE=REJECTED")
}

func data(s *session, params []string) {
	if len(params) != 2 {
		log.Fatal("invalid input, shouldn't happen")
	}

	token := params[0]

	if len(s.tx.rcptTo) > 0 {
		s.tx.archive.Meta("TO=" + strings.Join(s.tx.rcptTo[:], ","))
	}

	if version < "0.5" {
		fmt.Printf("filter-result|%s|%s|proceed\n", token, s.id)
	} else {
		fmt.Printf("filter-result|%s|%s|proceed\n", s.id, token)
	}

}

func dataLine(s *session, params []string) {
	if len(params) < 2 {
		log.Fatal("invalid input, shouldn't happen")
	}

	token := params[0]
	line := strings.Join(params[1:], "|")

	// Input is raw SMTP data - unescape leading dots and write to archive
	s.tx.archive.Data(strings.TrimPrefix(line, "."))

	//just relay the line as received
	if version < "0.5" {
		fmt.Printf("filter-dataline|%s|%s|%s\n", token, s.id, line)
	} else {
		fmt.Printf("filter-dataline|%s|%s|%s\n", s.id, token, line)
	}

}

func filterInit() {
	for k := range reporters {
		fmt.Printf("register|report|smtp-in|%s\n", k)
	}
	for k := range filters {
		fmt.Printf("register|filter|smtp-in|%s\n", k)
	}
	fmt.Println("register|ready")
}

func writeLine(s *session, token string, line string) {
	prefix := ""
	// Output raw SMTP data - escape leading dots.
	if strings.HasPrefix(line, ".") {
		prefix = "."
	}
	if version < "0.5" {
		fmt.Printf("filter-dataline|%s|%s|%s%s\n", token, s.id, prefix, line)
	} else {
		fmt.Printf("filter-dataline|%s|%s|%s%s\n", s.id, token, prefix, line)
	}
}

func trigger(actions map[string]func(*session, []string), atoms []string) {
	if atoms[4] == "link-connect" {
		// special case to simplify subsequent code
		s := session{}
		s.id = atoms[5]
		sessions[s.id] = &s
	}

	s := sessions[atoms[5]]
	if v, ok := actions[atoms[4]]; ok {
		v(s, atoms[6:])
	} else {
		os.Exit(1)
	}
}

func skipConfig(scanner *bufio.Scanner) {
	for {
		if !scanner.Scan() {
			os.Exit(0)
		}
		line := scanner.Text()
		if line == "config|ready" {
			return
		}
	}
}

func main() {

	fArg := flag.Bool("f",false,"Use flat filesystem path storage instead of <path>/YYYY-MM/DD")
	flag.Parse()

	if flag.Arg(0) == "" {
		log.Fatal("Archive storage path not given as last parameter (e.g. /var/db/mail-archive)")
		os.Exit(0)
	}

	archiveStorageFlat = *fArg;
	archiveStoragePath = flag.Arg(0)

	scanner := bufio.NewScanner(os.Stdin)

	skipConfig(scanner)

	filterInit()

	for {
		if !scanner.Scan() {
			os.Exit(0)
		}

		atoms := strings.Split(scanner.Text(), "|")
		if len(atoms) < 6 {
			os.Exit(1)
		}

		version = atoms[1]

		switch atoms[0] {
		case "report":
			trigger(reporters, atoms)
		case "filter":
			trigger(filters, atoms)
		default:
			os.Exit(1)
		}
	}
}
