package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/go-gandi/go-gandi"
	"github.com/go-gandi/go-gandi/config"
	"github.com/go-gandi/go-gandi/livedns"
	"github.com/rdegges/go-ipify"
)

type gandiConfig struct {
	APIKey      string
	SharingID   string
	Domain      string
	RecordNames string
}

func mustGetConfig() gandiConfig {
	config := gandiConfig{
		APIKey:      os.Getenv("GANDI_APIKEY"),
		SharingID:   os.Getenv("GANDI_SHARINGID"),
		Domain:      os.Getenv("GANDI_DOMAIN"),
		RecordNames: os.Getenv("GANDI_RECORDNAME"),
	}
	if config.APIKey == "" {
		log.Fatal("Environment variable GANDI_APIKEY must be set with the API key from Gandi")
	}
	if config.SharingID == "" {
		log.Fatal("Environment variable GANDI_SHARINGID must be set with the corporate uuid from Gandi")
	}
	if config.Domain == "" {
		log.Fatal("Environment variable GANDI_DOMAIN must be set with the domain you'd like to target")
	}
	if config.RecordNames == "" {
		log.Fatal("Environment variable GANDI_RECORDNAME must be set with the DNS records you'd like to update (A single entry), comma separated")
	}
	return config
}

func main() {
	dryRun := flag.Bool("n", false, "dry run")
	flag.Parse()
	gandiConfig := mustGetConfig()
	ip, err := ipify.GetIp()
	if err != nil {
		log.Fatal(err)
	}
	config := config.Config{
		APIKey:    gandiConfig.APIKey,
		SharingID: gandiConfig.SharingID,
	}
	dnsClient := gandi.NewLiveDNSClient(config)
	splitRecordNames := strings.Split(gandiConfig.RecordNames, ",")
	for _, recordName := range splitRecordNames {
		record, err := dnsClient.GetDomainRecordByNameAndType(gandiConfig.Domain, recordName, "A")
		if err != nil {
			log.Fatal(err)
		}
		if len(record.RrsetValues) != 1 {
			log.Fatalf("unexpected configuration of %s.%s: found %d record values instead of 1\n", recordName, gandiConfig.Domain, len(record.RrsetValues))
		}
		if ip != record.RrsetValues[0] {
			log.Printf("[%s %s] new IP found, before was %s, now is %s\n", recordName, gandiConfig.Domain, record.RrsetValues[0], ip)
			if !*dryRun {
				newRecords := []livedns.DomainRecord{}
				newRecords = append(newRecords, record)
				newRecords[0].RrsetValues[0] = ip
				response, err := dnsClient.UpdateDomainRecordsByName(gandiConfig.Domain, recordName, newRecords)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("[%s %s] IP changed to %s, with the following message: %s. You need to wait at least %d seconds for DNS change.\n", recordName, gandiConfig.Domain, ip, response.Message, record.RrsetTTL)
			} else {
				log.Printf("[%s %s] Dry run: IP not changed to %s. You would have waited for at least %d seconds for DNS change.\n", recordName, gandiConfig.Domain, ip, record.RrsetTTL)
			}
		} else {
			log.Printf("[%s %s] no change, still %s\n", recordName, gandiConfig.Domain, ip)
		}
	}
}
