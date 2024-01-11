package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context, event interface{}) (string, error) {
	fmt.Println("event", event)

	return "Hello world" + time.Now().GoString(), nil
}

func main() {
	lambda.Start(HandleRequest)
}
