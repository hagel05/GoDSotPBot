package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// Secrets represents the structure of the secrets stored in Secrets Manager
type Secrets struct {
	SlackBotToken          string `json:"slackBotToken"`
	GoBotVerificationToken string `json:"goBotVerificationToken"`
}

func getSecrets(secretName string) (Secrets, error) {
	// Create a Secrets Manager client
	// Create a Secrets Manager client with a specific region
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	secretsManagerClient := secretsmanager.New(sess)

	// Input to retrieve secret
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	// Retrieve secret value
	result, err := secretsManagerClient.GetSecretValue(input)
	if err != nil {
		return Secrets{}, fmt.Errorf("failed to retrieve secret value: %v", err)
	}

	// Check if SecretString or SecretBinary is present
	if result.SecretString == nil && result.SecretBinary == nil {
		return Secrets{}, fmt.Errorf("secret value is empty")
	}

	// Parse secret value
	var secrets Secrets
	if result.SecretString != nil {
		err = json.Unmarshal([]byte(*result.SecretString), &secrets)
		if err != nil {
			return Secrets{}, fmt.Errorf("failed to parse secret value: %v", err)
		}
	}

	return secrets, nil
}
