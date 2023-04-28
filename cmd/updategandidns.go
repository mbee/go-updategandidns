package main

import (
	"log"
	"os"

	"github.com/go-gandi/go-gandi"
	"github.com/go-gandi/go-gandi/config"
	"github.com/go-gandi/go-gandi/livedns"
	"github.com/rdegges/go-ipify"
)

type gandiConfig struct {
	APIKey     string
	SharingID  string
	Domain     string
	RecordName string
}

func mustGetConfig() gandiConfig {
	config := gandiConfig{
		APIKey:     os.Getenv("GANDI_APIKEY"),
		SharingID:  os.Getenv("GANDI_SHARINGID"),
		Domain:     os.Getenv("GANDI_DOMAIN"),
		RecordName: os.Getenv("GANDI_RECORDNAME"),
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
	if config.RecordName == "" {
		log.Fatal("Environment variable GANDI_RECORDNAME must be set with the DNS record you'd like to target (A single entry)")
	}
	return config
}

func main() {
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
	record, err := dnsClient.GetDomainRecordsByName(gandiConfig.Domain, gandiConfig.RecordName)
	if err != nil {
		log.Fatal(err)
	}
	if len(record) != 1 {
		log.Fatalf("unexpected configuration of %s.%s: found %d records instead of 1\n", gandiConfig.RecordName, gandiConfig.Domain, len(record))
	}
	if record[0].RrsetType != "A" {
		log.Fatalf("DNS Entry type for %s.%s must be A, found %s instead\n", gandiConfig.RecordName, gandiConfig.Domain, record[0].RrsetType)
	}
	if len(record[0].RrsetValues) != 1 {
		log.Fatalf("unexpected configuration of %s.%s: found %d record values instead of 1\n", gandiConfig.RecordName, gandiConfig.Domain, len(record[0].RrsetValues))
	}
	if ip != record[0].RrsetValues[0] {
		log.Printf("new IP found, before was %s, now is %s\n", record[0].RrsetValues[0], ip)
		newRecords := []livedns.DomainRecord{}
		newRecords = append(newRecords, record[0])
		newRecords[0].RrsetValues[0] = ip
		response, err := dnsClient.UpdateDomainRecordsByName(gandiConfig.Domain, gandiConfig.RecordName, newRecords)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("IP changed to %s, with the following message: %s. You need to wait at least %d seconds for DNS change.\n", ip, response.Message, record[0].RrsetTTL)
	} else {
		log.Println("no change")
	}
}
