package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/xuroi/xuroi/api/internal/email"
	"github.com/xuroi/xuroi/api/internal/notify"
	"github.com/xuroi/xuroi/api/internal/site"
)

func main() {
	to := flag.String("to", "", "recipient email")
	only := flag.String("only", "", "send one template: thread_reply, magic_link, password_reset")
	flag.Parse()
	if *to == "" {
		log.Fatal("usage: emailtest -to you@example.com")
	}

	siteCfg := site.Load()
	emailCfg := email.MergeSiteDefaults(email.ConfigFromEnv(), siteCfg)
	mailer, err := email.NewMailer(emailCfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	siteURL := siteCfg.Site.URL
	tests := []struct {
		name string
		send func() error
	}{
		{
			"magic_link",
			func() error {
				n := notify.New(nil, mailer, emailCfg, siteCfg, nil)
				return n.SendMagicLink(ctx, *to, "Doug", siteURL+"/auth/magic?token=TEST_TOKEN")
			},
		},
		{
			"password_reset",
			func() error {
				n := notify.New(nil, mailer, emailCfg, siteCfg, nil)
				return n.SendPasswordReset(ctx, *to, "Doug", siteURL+"/reset-password?token=TEST_TOKEN")
			},
		},
		{
			"thread_reply",
			func() error {
				communityName := siteCfg.Email.FromName
				if communityName == "" {
					communityName = siteCfg.Site.Name + " Community"
				}
				unsubURL := siteURL + "/email/unsubscribe?token=SES_DEMO_UNSUB_TOKEN"
				subject, htmlBody, textBody, err := email.RenderThreadReply(email.ThreadReplyData{
					CommunityName:    communityName,
					SiteURL:          siteURL,
					LogoURL:          email.LogoURL(siteURL),
					Recipient:        "Doug",
					IntroLine:        email.BuildIntroLine(1, "gear_tester", communityName),
					ThreadTitle:      "Best grip for humid climates?",
					ThreadURL:        siteURL + "/t/best-grip-for-humid-climates",
					CommunityURL:     siteURL + "/community",
					WatchedURL:       siteURL + "/community",
					DisableAllURL:    siteURL + "/settings/email",
					UnsubscribeURL:   unsubURL,
					Copyright:        "© 2006–2026 PutterTalk LLC.",
					ReplyCount:       1,
					Posts: []email.ThreadReplyPost{{
						Author:  "gear_tester",
						Excerpt: "I switched to a midsize Winn in Florida — night and day difference when it's 90% humidity.",
						When:    "Jul 2, 7:30 AM",
					}},
				})
				if err != nil {
					return err
				}
				return mailer.Send(ctx, email.Message{
					To:                 *to,
					Subject:            subject,
					HTMLBody:           htmlBody,
					TextBody:           textBody,
					ReplyTo:            siteCfg.Email.ReplyTo,
					MessageType:        "thread_reply",
					ListUnsubscribeURL: unsubURL,
				})
			},
		},
	}

	var failed int
	run := tests
	if *only != "" {
		var filtered []struct {
			name string
			send func() error
		}
		for _, t := range tests {
			if t.name == *only {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			log.Fatalf("unknown template %q", *only)
		}
		run = filtered
	}
	for _, t := range run {
		if err := t.send(); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", t.name, err)
			failed++
		} else {
			fmt.Printf("OK %s -> %s\n", t.name, *to)
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}