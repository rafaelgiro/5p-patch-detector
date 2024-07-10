package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	nc, err := nats.Connect("tls://connect.ngs.global", nats.UserCredentials("./NGS-Default-CLI.creds"), nats.Name("Patches"))
	if err != nil {
		log.Fatal(err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal(err)
	}

	live, pbe, err := getVersions()
	if err != nil {
		log.Fatal(err)
	}

	kv, err := js.KeyValue(ctx, "patches")

	lcurr, err := kv.Get(ctx, "live")
	if err != nil {
		log.Fatal(err)
	}

	pcurr, err := kv.Get(ctx, "pbe")
	if err != nil {
		log.Fatal(err)
	}

	if string(lcurr.Value()) != live {
		fmt.Println("Found new Live Version: ", live)
		_, err = kv.PutString(ctx, "live", live)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Still same Live version. Skipping...")
	}

	if string(pcurr.Value()) != pbe {
		fmt.Println("Found new PBE Version: ", pbe)
		_, err = kv.PutString(ctx, "pbe", pbe)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Still same PBE version. Skipping...")
	}
}

func getVersions() (string, string, error) {
	lresp, err := http.Get("https://raw.communitydragon.org/latest/content-metadata.json")
	if err != nil {
		return "", "", err
	}
	defer lresp.Body.Close()

	presp, err := http.Get("https://raw.communitydragon.org/pbe/content-metadata.json")
	if err != nil {
		return "", "", err
	}
	defer presp.Body.Close()

	lbody, err := io.ReadAll(lresp.Body)
	pbody, err := io.ReadAll(presp.Body)

	live := make(map[string]string)
	pbe := make(map[string]string)

	if err := json.Unmarshal(lbody, &live); err != nil {
		return "", "", err
	}

	if err := json.Unmarshal(pbody, &pbe); err != nil {
		return "", "", err
	}

	return live["version"], pbe["version"], nil

}
