package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context) (*string, error) {
	nc, err := nats.Connect("tls://connect.ngs.global", nats.UserCredentials("./NGS-Default-CLI.creds"), nats.Name("Patches"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	live, pbe, err := getVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	kv, err := js.KeyValue(ctx, "patches")
	if err != nil {
		return nil, fmt.Errorf("failed to get KeyValue store: %w", err)
	}

	lcurr, err := kv.Get(ctx, "live")
	if err != nil {
		return nil, fmt.Errorf("failed to get current live version: %w", err)
	}

	pcurr, err := kv.Get(ctx, "pbe")
	if err != nil {
		return nil, fmt.Errorf("failed to get current PBE version: %w", err)
	}

	if string(lcurr.Value()) != live {
		fmt.Println("Found new Live Version: ", live)
		_, err = kv.PutString(ctx, "live", live)
		if err != nil {
			return nil, fmt.Errorf("failed to update live version: %w", err)
		}
	} else {
		fmt.Println("Still same Live version. Skipping...")
	}

	if string(pcurr.Value()) != pbe {
		fmt.Println("Found new PBE Version: ", pbe)
		_, err = kv.PutString(ctx, "pbe", pbe)
		if err != nil {
			return nil, fmt.Errorf("failed to update PBE version: %w", err)
		}
	} else {
		fmt.Println("Still same PBE version. Skipping...")
	}

	msg := fmt.Sprint("Success!")

	return &msg, nil
}

func getVersions() (string, string, error) {
	lresp, err := http.Get("https://raw.communitydragon.org/latest/content-metadata.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to get live version: %w", err)
	}
	defer lresp.Body.Close()

	presp, err := http.Get("https://raw.communitydragon.org/pbe/content-metadata.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to get PBE version: %w", err)
	}
	defer presp.Body.Close()

	lbody, err := io.ReadAll(lresp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read live response body: %w", err)
	}

	pbody, err := io.ReadAll(presp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read PBE response body: %w", err)
	}

	live := make(map[string]string)
	pbe := make(map[string]string)

	if err := json.Unmarshal(lbody, &live); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal live version: %w", err)
	}

	if err := json.Unmarshal(pbody, &pbe); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal PBE version: %w", err)
	}

	return live["version"], pbe["version"], nil

}
