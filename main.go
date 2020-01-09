package main

import (
	"fmt"
	"log"
	"time"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"os"
	"os/exec"
	"bytes"
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

type IPFSClusterAddResponse struct {
	Name string `json:"name""`
	Cid KV `json:"cid"`
	Size int64 `json:"size"`
}

type IPFSCluster struct {
	host string
	basicAuth string
}

type IPFSClusterResponseCid struct {
	Path string
	Cid string
}

func NewIPFSCluster() (*IPFSCluster) {
	return &IPFSCluster{
		host: "/dnsaddr/cluster.ipfs.io",
		//		basicAuth: os.Getenv("IPFS_CLUSTER_AUTH")}
		basicAuth: ""}
}

func (c *IPFSCluster) Add(archivePath string) ([]string, error) {
	cids := make([]string, 0)
	addCmd:= exec.Command("ipfs-cluster-ctl", "--basic-auth", c.basicAuth, "--host", c.host, "--enc=json", "add", "-w", "--no-stream", archivePath)
	//DEBUG
	log.Printf("Adding to cluster: %s", archivePath)
	stdout, err := addCmd.StdoutPipe()
	if err != nil {
		return cids, fmt.Errorf("add to cluster: %s: %v", archivePath , err)
	}

	if err := addCmd.Start(); err != nil {
		return cids, fmt.Errorf("add to cluster: %s: %v", archivePath , err)
	}

	result := new(bytes.Buffer)
	result.ReadFrom(stdout)

	if err := addCmd.Wait(); err != nil {
		return cids, fmt.Errorf("add to cluster: %s: %v", archivePath , err)
	}

	log.Printf("Result of adding %s: %s", archivePath, result)
	addResponses := make([]IPFSClusterAddResponse, 0)
	err = json.Unmarshal(result.Bytes(), &addResponses)
	if err != nil {
		return cids, fmt.Errorf("add to cluster: %s: %v", archivePath , err)
}
	log.Printf("addResponses: %v", addResponses)

	// Extract CIDs from response
	// Can't use json.Unmarshall() because the keys of the JSON response are variable
	for _, response := range addResponses {
		for _, cid := range response.Cid {
			cids = append(cids, cid)
		}
	}
	return cids, nil
}

func (c *IPFSCluster) Pin(cids []string) (error) {
	for _, cid := range cids {
		pinCmd:= exec.Command("ipfs-cluster-ctl", "--basic-auth", c.basicAuth, "--host", c.host, "--enc=json", "pin", "add", cid)
		// NOTE `ipfs-cluster-ctl pin add` exits without error regardless of whether the CID exists
		log.Printf("Pinning %s", cid)
		err := pinCmd.Run()
		if err != nil {
			return fmt.Errorf("pin %s: %v", cid, err)
		}
	}
	return nil
}

func receive(rw http.ResponseWriter, req *http.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		//panic(err)
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}
	//fmt.Println(string(body))
	var t Data
	fmt.Printf("body: %s", body)
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
	fmt.Printf("Alerts: %v", t.Alerts)

	for _ , v := range t.Alerts {
		if (v.Labels["alertname"] == "node_high_memory_usage_95_percent") {
			pprofs, err := NewPprofRequest(v.Labels["instance"], os.Getenv("PPROF_AUTH_PASS"))
			if err != nil {
				log.Printf("Error: %v\n", err)
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				break
			}

			_, err = pprofs.Collect()
			if err != nil {
				log.Printf("Error: %v\n", err)
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				break
			}

			//log.Printf("Created pprof archive: %s\n", archivePath)
		}
	}
}

func main() {
	if len(os.Getenv("PPROF_AUTH_PASS")) == 0 {
		log.Fatal("Missing HTTP Basic Auth password. Please Set/Export PPROF_AUTH_PASS")
	}

	http.HandleFunc("/", receive)
	log.Fatal(http.ListenAndServe(":9096", nil))
}
