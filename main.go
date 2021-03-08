package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
)

var webhooks = make(map[string]string)

func main() {
	domain := os.Getenv("DOMAIN")
	code := os.Getenv("CODE")

	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		if strings.HasPrefix(kv[0], "WEBHOOK_") == true {
			key := code + "+" + strings.ToLower(kv[0][8:]) + "@"
			value := kv[1]
			webhooks[key] = value

			log.Printf("Forwarding %s%s to %s\n", key, domain, value)
		}
	}

	s := smtp.NewServer(&Backend{})

	s.Addr = "[::]:25"
	s.Domain = domain
	s.AllowInsecureAuth = true
	s.AuthDisabled = true
	s.EnableSMTPUTF8 = false

	s.ListenAndServe()
}

type Backend struct{}

func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &Session{}, nil
}

func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{}, nil
}

type Session struct {
	WebhookURL string
	Debug      bool
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	log.Println("From:", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Println("To:", to)

	e, err := mail.ParseAddress(to)
	if err != nil {
		log.Println(err)
		return err
	}

	if strings.HasPrefix(e.Address, "postmaster@") {
		s.Debug = true
		return nil
	}

	if strings.HasPrefix(e.Address, "abuse@") {
		s.Debug = true
		return nil
	}

	for prefix, url := range webhooks {
		if strings.HasPrefix(e.Address, prefix) {
			s.WebhookURL = url
			return nil
		}
	}

	log.Println("No mailbox", to)

	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      "No mailbox",
	}
}

func (s *Session) Data(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		log.Println(err)
		return err
	}

	if s.Debug == true {
		log.Println(string(buf))
	}

	if s.WebhookURL == "" {
		return nil
	}

	log.Println("POST", s.WebhookURL)
	resp, err := http.Post(s.WebhookURL, "message/rfc822", bytes.NewReader(buf))
	if err != nil {
		log.Println(err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Println(resp)
		return &smtp.SMTPError{
			Code:         450,
			EnhancedCode: smtp.EnhancedCode{4, 5, 0},
			Message:      "Failed to relay message",
		}
	}

	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
