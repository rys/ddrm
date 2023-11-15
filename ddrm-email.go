//go:build client
// +build client

package main

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/matcornic/hermes/v2"
)

func sendTestEmail() {
	if stateSendTestEmail {
		mockFetched := []string{"1.1.1.1"}
		mockCached := []string{"2.2.2.2"}
		sent := sendEmail("example.com", ddrmRecordTypeA, mockFetched, mockCached)
		if sent {
			dbgf(ddrmSuccessSentMail)
		} else {
			dbgf(ddrmErrorSendingMail)
		}
		os.Exit(ddrmExitAfterMailTest)
	}
}

// try and send an email report with records
// it's not defensive and will just return to the caller with nil if sending fails
func sendEmail(fqdn string, recordType DdrmRecordType, fetched []string, cached []string) (sent bool) {
	sent = false

	now := time.Now()

	hermesMailer := hermes.Hermes{
		Product: hermes.Product{
			Name:      ddrmAppConfig.EmailSenderName,
			Link:      ddrmAppConfig.EmailLink,
			Logo:      ddrmAppConfig.EmailLogo,
			Copyright: "Copyright (c) " + now.Format("2006") + " " + ddrmAppConfig.EmailSenderName,
		},
	}

	// prepare a hermes.Entry record for the data table
	entry := [][]hermes.Entry{
		{
			{Key: "FQDN", Value: fqdn},
			{Key: "Record", Value: string(recordType)},
			{Key: "Expected", Value: strings.Join(cached, ", ")},
			{Key: "Currently", Value: strings.Join(fetched, ", ")},
		},
	}

	// prepare a hermes.Email object with configured Body, Intros, Outros and the data table for record showing
	email := hermes.Email{
		Body: hermes.Body{
			Name: ddrmAppConfig.EmailToName,
			Intros: []string{
				ddrmAppConfig.EmailSenderName + " has detected a record change.",
			},
			Outros: []string{
				"The detection was recorded on " + now.Format(time.RFC1123),
				"Detected by " + fmt.Sprintf(ddrmStartupBanner, BuildVersion, BuildDate, GitRev, BuildUser),
			},
			Table: hermes.Table{
				Data: entry,
				Columns: hermes.Columns{
					CustomWidth: map[string]string{
						"FQDN":      "15%",
						"Record":    "15%",
						"Expected":  "35%",
						"Currently": "35%",
					},
				},
			},
		},
	}

	emailBody, err := hermesMailer.GenerateHTML(email)

	if statePlainTextEmail {
		emailBody, err = hermesMailer.GeneratePlainText(email)
	}

	if err != nil {
		dbg(ddrmErrorUnableToGenerateEmail)
		return
	}

	dateString := fmt.Sprintf("Date: %s\r\n", now.Format(time.RFC1123))

	to := []string{
		ddrmAppConfig.EmailTo,
	}

	envelope := []byte(
		"To: " + ddrmAppConfig.EmailToName + "<" + ddrmAppConfig.EmailTo + ">\r\n" +
			"From: " + ddrmAppConfig.EmailUserName + "<" + ddrmAppConfig.EmailUser + ">\r\n" +
			"Subject: " + ddrmAppConfig.EmailSubject + "\r\n" +
			"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n" +
			dateString + "\r\n")

	message := append(envelope, []byte(emailBody)...)

	auth := smtp.PlainAuth("", ddrmAppConfig.EmailUser, ddrmAppConfig.EmailPassword, ddrmAppConfig.EmailServerHostname)
	err = smtp.SendMail(ddrmAppConfig.EmailServerHostname+":"+ddrmAppConfig.EmailServerPort, auth, ddrmAppConfig.EmailUser, to, message)

	if err != nil {
		dbg(ddrmErrorSendingMail)
	} else {
		sent = true
		dbg(ddrmSuccessSentMail)
	}

	return
}
