package main

import (
	"net/http"
	"encoding/json"
	"time"
	"io/ioutil"
	"log"
	"fmt"
	"os"
	"os/exec"
)

type PprofRequest struct {
	Instance string
	netClient *http.Client
	tempDir string
	profiles []Profile
	IpfsVersion IPFSVersion
}

type Profile struct {
	url string
	fileName string
	svg bool
}

type IPFSVersion struct {
	Version string `json:"Version"`
	Commit string `json:"Commit"`
	Repo string `json:"Repo"`
	System string `json:"System"`
	Golang string `json:"Golang"`
	//{"Version":"0.5.0-dev","Commit":"ae0e31d","Repo":"7","System":"amd64/linux","Golang":"go1.12.9"}
}

func NewPprofRequest(instance string) *PprofRequest {
	netClient := &http.Client{
		Timeout: time.Second * 120,
	}
	tempDir, err := ioutil.TempDir("", "example")
	if err != nil {
		log.Fatal(err)
	}
	// DEBUG
	log.Printf("temp dir: %v", tempDir)

	profiles := []Profile{{url: "/debug/pprof/goroutine?debug=2"}}

	request := PprofRequest{
		Instance: instance + ".dwebops.net",
		netClient: netClient,
		tempDir: tempDir,
		profiles: profiles}

	return &request
}

//
// func (r *PprofRequest) Collect() {
// 	for _, req
// }

func (r *PprofRequest) Collect() (string){
	log.Printf("Collecting pprofs for %s to %s", r.Instance, r.tempDir)
	err := r.fetchVersion()
	if err != nil { log.Fatal(err) }
	log.Printf("Instance %s running version: %s-%s", r.Instance, r.IpfsVersion.Version, r.IpfsVersion.Commit)
	r.goroutineStacks()
	r.goroutineProfile()
	r.heapProfile()
	r.cpuProfile()
	r.mutexProfile()
	archivePath, err := r.createArchive()
	if err != nil { log.Fatal(err) }
	return archivePath
}

func (r *PprofRequest) goroutineStacks() {
	profile, err := r.fetchPprof("/debug/pprof/goroutine?debug=2", "goroutine.stacks")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
}

func (r *PprofRequest) goroutineProfile() {
	profile, err := r.fetchPprof("/debug/pprof/goroutine", "goroutine.pprof")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
	generateSVG(profile)
}

func (r *PprofRequest) heapProfile() {
	profile, err := r.fetchPprof("/debug/pprof/heap", "heap.pprof")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
	generateSVG(profile)
}

// Collecting cpu profile (~30s)
func (r *PprofRequest) cpuProfile() {
	profile, err := r.fetchPprof("/debug/pprof/profile", "cpuProfile.pprof")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
	generateSVG(profile)
}

func (r *PprofRequest) mutexProfile() {
	// Enabling mutex profiling"
	postURLEnable := fmt.Sprintf("https://%s%s", r.Instance, "/debug/pprof-mutex/?fraction=4")
	postReqEnable, err := http.NewRequest("POST", postURLEnable, nil)
	postReqEnable.SetBasicAuth("admin", os.Getenv("PPROF_AUTH_PASS"))
	//DEBUG
	// log.Println(postReqEnable)
	resp, err := r.netClient.Do(postReqEnable)
	if err != nil { log.Fatal(err) }
	//DEBUG
	log.Println(resp)

	// Waiting for mutex data to be updated
	time.Sleep(30 * time.Second)

	debug, err := r.fetchPprof("/debug/pprof/mutex?debug=2", "mutexDebug.txt")
	if err != nil { log.Fatal(err) }
	log.Println(debug)

	profile, err := r.fetchPprof("/debug/pprof/mutex", "mutexProfile.pprof")
	if err != nil { log.Fatal(err) }
	log.Println(profile)

	// Disabling mutex profiling"
	postURLDisable := fmt.Sprintf("https://%s%s", r.Instance, "/debug/pprof-mutex/?fraction=0")
	postReqDisable, err := http.NewRequest("POST", postURLDisable, nil)
	postReqDisable.SetBasicAuth("admin", os.Getenv("PPROF_AUTH_PASS"))
	//DEBUG
	// log.Println(postReqDisable)
	resp, err = r.netClient.Do(postReqDisable)
	if err != nil { log.Fatal(err) }
	// DEBUG
	// log.Println(resp)

	generateSVG(profile)
}

func (r *PprofRequest) fetchPprof(location string, localFilename string) (string, error) {
	url := fmt.Sprintf("https://%s%s", r.Instance, location)
	fileLocation := r.tempDir + "/" + localFilename
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("admin", os.Getenv("PPROF_AUTH_PASS"))

	log.Printf("fetching %s profile from %s", localFilename, url)
	resp, err := r.netClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	// log.Printf("resp: %v", resp)
	body, err := ioutil.ReadAll(resp.Body)
	err = ioutil.WriteFile(fileLocation, body, 0644)
	if err != nil {
		return "", err
	}

	return fileLocation, nil
}

func (r *PprofRequest) fetchVersion() (error) {
	url := fmt.Sprintf("http://%s%s", r.Instance, "/api/v0/version")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil { return err }

	resp, err := r.netClient.Do(req)
	defer resp.Body.Close()
	if err != nil { return err }

	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &r.IpfsVersion)
	if err != nil { return err }

	return nil
}

	func (r *PprofRequest) createArchive() (string, error) {
	archivePath := fmt.Sprintf("/tmp/%s_%s-%s.tar.gz", r.Instance, r.IpfsVersion.Version, r.IpfsVersion.Commit)
	tarCmd:= exec.Command("tar", "czf", archivePath, r.tempDir)
	err := tarCmd.Run()
	if err != nil {
		return "", err
	}
	log.Printf("Generated %s", archivePath)
	return archivePath, nil
}


func generateSVG(profilePath string) {
	svgOutput := profilePath + ".svg"
	goToolCmd:= exec.Command("go", "tool", "pprof", "-symbolize=remote", "-svg", "-output", svgOutput, profilePath)
	err := goToolCmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Generated %s", svgOutput)
}
