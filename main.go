package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
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
	if s.WebhookURL != "" {
		json, err := eml2json(r)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Println("POST", s.WebhookURL)
		resp, err := http.Post(s.WebhookURL, "application/json", bytes.NewReader(json))
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
	}

	if s.Debug == true {
		buf, err := ioutil.ReadAll(r)
		if err == nil {
			log.Println(string(buf))
		}
	}

	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

type Email struct {
	From        Address           `json:"from,omitempty"`
	To          []Address         `json:"to,omitempty"`
	CC          []Address         `json:"cc,omitempty"`
	BCC         []Address         `json:"bcc,omitempty"`
	Date        string            `json:"date,omitempty"`
	Subject     string            `json:"subject,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	BodyText    string            `json:"bodyText,omitempty"`
	BodyHTML    string            `json:"bodyHTML,omitempty"`
	Attachments []Part            `json:"attachments,omitempty"`
	Inlines     []Part            `json:"inlines,omitempty"`
}

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type Part struct {
	ContentType string `json:"contentType,omitempty"`
	FileName    string `json:"filename,omitempty"`
	Content     string `json:"content,omitempty"`
}

func eml2json(r io.Reader) ([]byte, error) {
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}

	m := Email{}

	alist, err := env.AddressList("From")
	if err == nil {
		for _, addr := range alist {
			m.From = Address{addr.Name, addr.Address}
			break
		}
	}

	alist, err = env.AddressList("To")
	if err == nil {
		for _, addr := range alist {
			m.To = append(m.To, Address{addr.Name, addr.Address})
		}
	}

	alist, err = env.AddressList("CC")
	if err == nil {
		for _, addr := range alist {
			m.To = append(m.CC, Address{addr.Name, addr.Address})
		}
	}

	alist, err = env.AddressList("BCC")
	if err == nil {
		for _, addr := range alist {
			m.To = append(m.BCC, Address{addr.Name, addr.Address})
		}
	}

	m.Date = env.GetHeader("Date")
	m.Subject = env.GetHeader("Subject")

	m.Headers = make(map[string]string)
	headers := env.GetHeaderKeys()
	for _, header := range headers {
		m.Headers[header] = env.GetHeader(header)
	}

	m.BodyText = env.Text
	m.BodyHTML = env.HTML

	for _, part := range env.Attachments {
		m.Attachments = append(m.Attachments, Part{part.ContentType, part.FileName, string(part.Content)})
	}

	for _, part := range env.Inlines {
		m.Inlines = append(m.Inlines, Part{part.ContentType, part.FileName, string(part.Content)})

	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}
