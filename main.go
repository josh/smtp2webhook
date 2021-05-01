package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/namsral/flag"
)

const (
	name      = "smtp2webhook"
	envPrefix = "SMTP2WEBHOOK"
	version   = "1.1.0"
)

var (
	fs           *flag.FlagSet
	domain       string
	code         string
	healthcheck  bool
	printVersion bool
)

var webhooks = make(map[string]string)

func main() {
	fs = flag.NewFlagSetWithEnvPrefix(name, envPrefix, flag.ExitOnError)
	fs.StringVar(&domain, "domain", "localhost", "domain")
	fs.StringVar(&code, "code", "", "secret code")
	fs.BoolVar(&healthcheck, "healthcheck", false, "run healthcheck")
	fs.BoolVar(&printVersion, "version", false, "print version")
	fs.Parse(os.Args[1:])

	if printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		if strings.HasPrefix(kv[0], "SMTP2WEBHOOK_URL_") {
			key := code + "+" + strings.ToLower(kv[0][17:]) + "@"
			value := kv[1]
			webhooks[key] = value

			log.Printf("Forwarding %s%s to %s\n", key, domain, value)
		}
	}

	if healthcheck {
		client, err := smtp.Dial("127.0.0.1:25")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = client.Hello("localhost")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	s := smtp.NewServer(&Backend{})

	s.Addr = "[::]:25"
	s.Domain = domain
	s.AllowInsecureAuth = true
	s.AuthDisabled = true
	s.EnableSMTPUTF8 = false

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type Backend struct{}

func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &Session{}, nil
}

func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{}, nil
}

type Session struct {
	From       string
	To         string
	WebhookURL string
	Debug      bool
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.From = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	s.To = to

	e, err := mail.ParseAddress(to)
	if err != nil {
		log.Println(s.From, "->", s.To, "501")
		log.Println(err)
		return err
	}

	if strings.HasPrefix(e.Address, "postmaster@") || strings.HasPrefix(e.Address, "abuse@") {
		s.Debug = true
		return nil
	}

	for prefix, url := range webhooks {
		if strings.HasPrefix(e.Address, prefix) {
			s.WebhookURL = url
			return nil
		}
	}

	log.Println(s.From, "->", s.To, "550")
	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      "No mailbox",
	}
}

func (s *Session) Data(r io.Reader) error {
	log.Println(s.From, "->", s.To)

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		log.Println(err)
		return err
	}

	if s.Debug {
		log.Println(string(buf))
	}

	if s.WebhookURL == "" {
		return nil
	}

	resp, err := http.Post(s.WebhookURL, "message/rfc822", bytes.NewReader(buf))
	if err != nil {
		log.Println("POST", s.WebhookURL, err)
		return err
	}

	log.Println("POST", s.WebhookURL, resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	} else {
		return &smtp.SMTPError{
			Code:         450,
			EnhancedCode: smtp.EnhancedCode{4, 5, 0},
			Message:      "Failed to relay message",
		}
	}
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
