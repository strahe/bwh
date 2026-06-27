package client_test

import (
	"context"
	"log"

	"github.com/strahe/bwh/pkg/client"
)

func ExampleNewClient() {
	c := client.NewClient("your-api-key", "your-veid")
	ctx := context.Background()

	info, err := c.GetServiceInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("hostname: %s", info.Hostname)
}

func ExampleGetBWHError() {
	c := client.NewClient("your-api-key", "your-veid")
	ctx := context.Background()

	_, err := c.GetRateLimitStatus(ctx)
	if err != nil {
		if apiErr, ok := client.GetBWHError(err); ok {
			log.Printf("KiwiVM API error %d: %s", apiErr.Code, apiErr.Message)
			return
		}
		log.Fatal(err)
	}
}
