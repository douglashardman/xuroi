package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESMailer sends via Amazon SES v2.
type SESMailer struct {
	client *sesv2.Client
	cfg    Config
}

func NewSESMailer(cfg Config) (*SESMailer, error) {
	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return &SESMailer{client: sesv2.NewFromConfig(awsCfg), cfg: cfg}, nil
}

func (m *SESMailer) Send(ctx context.Context, msg Message) error {
	replyTo := msg.ReplyTo
	if replyTo == "" {
		replyTo = m.cfg.ReplyTo
	}
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(m.cfg.FromHeader()),
		Destination: &types.Destination{
			ToAddresses: []string{msg.To},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject), Charset: aws.String("UTF-8")},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(msg.HTMLBody), Charset: aws.String("UTF-8")},
					Text: &types.Content{Data: aws.String(msg.TextBody), Charset: aws.String("UTF-8")},
				},
			},
		},
	}
	if replyTo != "" {
		input.ReplyToAddresses = []string{replyTo}
	}
	if msg.ListUnsubscribeURL != "" {
		input.EmailTags = append(input.EmailTags, types.MessageTag{
			Name:  aws.String("xuroi_type"),
			Value: aws.String(msg.MessageType),
		})
		input.Content.Simple.Headers = append(input.Content.Simple.Headers,
			types.MessageHeader{
				Name:  aws.String("List-Unsubscribe"),
				Value: aws.String("<" + msg.ListUnsubscribeURL + ">"),
			},
			types.MessageHeader{
				Name:  aws.String("List-Unsubscribe-Post"),
				Value: aws.String("List-Unsubscribe=One-Click"),
			},
		)
	}
	_, err := m.client.SendEmail(ctx, input)
	return err
}