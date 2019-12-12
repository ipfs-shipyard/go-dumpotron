package main

import (
	"fmt"
	"log"
	"time"
	"io/ioutil"
	"net/http"
	"encoding/json"
)

// KV is a set of key/value string pairs.
type KV map[string]string

type Data struct {
	Version  string `json:"version"`
	GroupKey string `json:"groupKey"`
	Status   string `json:"status"`
	Receiver string `json:"receiver"`

	GroupLabels       KV `json:"groupLabels"`
	CommonLabels      KV `json:"commonLabels"`
	CommonAnnotations KV `json:"commonAnnotations"`

	ExternalURL string `json:"externalURL"`

	Alerts   Alerts `json:"alerts"`
}

// Alert holds one alert for notification templates.
type Alert struct {
	Status       string    `json:"status"`
	Labels       KV        `json:"labels"`
	Annotations  KV        `json:"annotations"`
	StartsAt     time.Time `json:"startsAt"`
	EndsAt       time.Time `json:"endsAt"`
	GeneratorURL string    `json:"generatorURL"`
	Fingerprint  string    `json:"fingerprint"`
}

// Alerts is a list of Alert objects.
type Alerts []Alert

type Message struct {
	*Data

	// The protocol version.
}

func receive(rw http.ResponseWriter, req *http.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		//panic(err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}
	//fmt.Println(string(body))
	var t Data
	err = json.Unmarshal(body, &t)
	if err != nil {
		//panic(err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	//fmt.Println(t)
	// DEBUG
	fmt.Println("GroupKey    :", t.GroupKey)
	fmt.Println("Receiver    :", t.Receiver)
	fmt.Println("Status      :", t.Status)
	fmt.Println("Version     :", t.Version)
	fmt.Println("Status      :", t.Status)
	fmt.Println("GroupLabels :", t.GroupLabels)
	fmt.Printf("Labels %v", t.Alerts)
	//fmt.Println("\tAlertName :", t.GroupLabels.AlertName)
	for k , v := range t.Alerts {
		fmt.Println("\t", k)
		fmt.Println("\t\tStatus      :" , v.Status)
		fmt.Println("\t\tStartsAt    :" ,v.StartsAt)
		fmt.Println("\t\tEndsAt      :"  ,v.EndsAt)
		fmt.Printf("\t\tGeneratorURL: %s\n ", v.GeneratorURL)
		fmt.Println("\t\tLabels:     :",v.Labels)
		fmt.Println("\t\t\tAlertname:        :",v.Labels["alertname"])
		fmt.Println("\t\t\tInstance:        :",v.Labels["instance"])
		//fmt.Println("\t\t\tDrp_service:      :",v.Labels.Drp_service)
		//fmt.Println("\t\t\tDrp_stage:        :",v.Labels.Drp_stage)
		//fmt.Println("\t\t\tDrp_vertical:     :",v.Labels.Drp_vertical)
		//fmt.Println("\t\t\tInstance:         :",v.Labels.Instance)
		//fmt.Println("\t\t\tJob:              :",v.Labels.Job)
		//fmt.Println("\t\t\tSeverity:         :",v.Labels.Severity)
		fmt.Println("\t\tAnnotations :", v.Annotations)
		// fmt.Println("\t\t\tDescription :", v.Annotations.Description)
		// fmt.Println("\t\t\tSummary     :", v.Annotations.Summary)

	}
}

func main() {
	pprofs := NewPprofRequest("gateway-bank2-ams1.dwebops.net")
	pprofs.Collect()
	http.HandleFunc("/", receive)
	log.Fatal(http.ListenAndServe(":9096", nil))
}
