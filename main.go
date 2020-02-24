package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"fmt"
	"github.com/termie/go-shutil"
	"time"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const GatewaysDomain = ".dwebops.net"

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

type Alerts []Alert

type Message struct {
	*Data
}

func receive(rw http.ResponseWriter, req *http.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}
	var t Data
	log.Debugf("body: %s", body)
	err = json.Unmarshal(body, &t)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	log.Debug("GroupKey    :", t.GroupKey)
	log.Debug("Receiver    :", t.Receiver)
	log.Debug("Status      :", t.Status)
	log.Debug("Version     :", t.Version)
	log.Debug("Status      :", t.Status)
	log.Debug("GroupLabels :", t.GroupLabels)
	log.Debug("Alerts: %v", t.Alerts)

	for _ , v := range t.Alerts {
		log.Infof("Received alert for instance %s: %s", v.Labels["instance"], v.Labels["alertname"])
		pprofs, err := NewPprofRequest(v.Labels["instance"] + GatewaysDomain, os.Getenv("PPROF_AUTH_PASS"))
		if err != nil {
		    log.Errorf("Error: %v\n", err)
		    http.Error(rw, err.Error(), http.StatusInternalServerError)
		    break
		}

		// collect pprof dumps archive
		archivePath, err := pprofs.Collect()
		defer cleanupTempFiles(archivePath)
		if err != nil {
		    log.Errorf("Error: %v\n", err)
		    http.Error(rw, err.Error(), http.StatusInternalServerError)
		    break
		}

		// add & pin archive to IPFS cluster
		cidURL, err := ipfsClusterClient.AddAndPin(archivePath)
		log.Infof("pinned archive URL: %s", cidURL)
		if err != nil {
		    log.Errorf("Error: %v\n", err)
		    http.Error(rw, err.Error(), http.StatusInternalServerError)
		    break
		}

		ipfsVersion := pprofs.ipfsVersion.String()
		// Fetch GH issue for go-ipfs version
		ghIssue, err := getGHIssue(ipfsVersion)
		log.Debugf("Found GH Issue for version %s: %s", ipfsVersion, *ghIssue.HTMLURL)
		if err != nil {
		    log.Errorf("Error: %v\n", err)
		    http.Error(rw, err.Error(), http.StatusInternalServerError)
		    break
		}

		// Post comment with pprof dump URL on GH issue
		commentURL, err := postArchiveCIDtoGH(cidURL, ghIssue)
		log.Infof("Added pprof dump URL to new comment at: %s", commentURL)
		if err != nil {
		    log.Errorf("Error: %v\n", err)
		    http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	}
}

func main() {
	daemon := flag.Bool("daemon", false, "a bool")
	flag.Parse()
	log.SetLevel(log.InfoLevel)
	if strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) == "debug" {
		log.SetLevel(log.DebugLevel)
	}

	if (*daemon == true) {
		startDaemon()
	} else {
		dumpLocally()
	}

}

func startDaemon() {
	checkEnvs([]string{"PPROF_AUTH_PASS", "IPFS_CLUSTER_AUTH", "GITHUB_TOKEN"})

	// Setup clients
	authToken := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	setupGHClient(authToken)
	ipfsClusterToken := strings.TrimSpace(os.Getenv("IPFS_CLUSTER_AUTH"))
	setupIPFSClusterClient(ipfsClusterToken)

	http.HandleFunc("/", receive)
	log.Infof("HTTP server started on port %d", 9096)
	log.Fatal(http.ListenAndServe(":9096", nil))
}

func dumpLocally() {
	checkEnvs([]string{"PPROF_AUTH_PASS"})
	if len(os.Args) < 2 {
		log.Fatal("Please specify instance address (eg: gateway-bank1-sjc1.dwebops.net)")
	}
	instance := os.Args[1]
	log.Infof("Dumping pprof locally for %s", instance)
	pprofs, err := NewPprofRequest(instance, os.Getenv("PPROF_AUTH_PASS"))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// collect pprof dumps archive
	archivePath, err := pprofs.Collect()
	defer cleanupTempFiles(archivePath)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Debugf("temp dump created at %s", archivePath)


	cwdArchivePath, err := copyArchiveToCwd(archivePath)
	if err != nil {
		log.Fatalf("Error moving archive to CWD: %v", err)
	}
	log.Infof("dump created at %s", cwdArchivePath)

}

func checkEnvs(envs []string) {
	for _, env := range envs {
		if len(os.Getenv(env)) == 0 {
			log.Fatalf("Please Set/Export env %s", env)
		}
	}
}

func copyArchiveToCwd(archivePath string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", nil
	}

	cwdArchivePath := dir + "/" + path.Base(archivePath)
	log.Debugf("Copying archive from %s => %s", archivePath, cwdArchivePath)
	fileCopiedTo , err := shutil.Copy(archivePath, cwdArchivePath, false)
	if err != nil { return "", fmt.Errorf("Error moving file from %s to %s: %v", archivePath, cwdArchivePath, err) }
	log.Debugf("File copied to: %s", fileCopiedTo)

	return cwdArchivePath, nil
}

func cleanupTempFiles(archivePath string) {
	tmpDir := filepath.Dir(archivePath)
	log.Debugf("Cleaning up tempfiles from %s", tmpDir)
	err := os.RemoveAll(tmpDir)
	if err != nil {
		log.Warnf("Error cleaning up temp files: %v", err)
	}
}
