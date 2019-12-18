package main

import (
	"net/http"
	"time"
	"io/ioutil"
	"log"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type PprofRequest struct {
	Instance string
	netClient *http.Client
	tempDir string
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

	request := PprofRequest{
		Instance: instance,
		netClient: netClient,
		tempDir: tempDir }

	return &request
	}

//
func (r *PprofRequest) Collect() {
	// r.goroutineStacks()
	r.goroutineProfile()
	// r.heapProfile()
	// r.cpuProfile()
	// r.mutexProfile()
}

func (r *PprofRequest) goroutineStacks() {
	profile, err := r.fetchPprof("/debug/pprof/goroutine?debug=2", "goroutine.stacks")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
	generateSVG(profile)

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
	// -svg -output heap.svg "http://$HTTP_API/debug/pprof/heap"
}

// Collecting cpu profile (~30s)
func (r *PprofRequest) cpuProfile() {
	profile, err := r.fetchPprof("/debug/pprof/profile", "cpuProfile.pprof")
	if err != nil { log.Fatal(err) }
	log.Println(profile)
	// go tool pprof -symbolize=remote -svg -output cpu.svg "http://$HTTP_API/debug/pprof/profile"
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

	// go tool pprof -symbolize=remote -svg -output mutex.svg "http://$HTTP_API/debug/pprof/mutex"
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


func generateSVG(profilePath string) {
	// go tool pprof -symbolize=remote -svg -output mutex.svg "http://$HTTP_API/debug/pprof/mutex"
	dirPath := filepath.Dir(profilePath)
	output := filepath.Join(dirPath, filepath.Base(profilePath) + ".svg")
	goToolCmd:= exec.Command("go", "tool", "pprof", "-symbolize=remote", "-svg", "-output", output, profilePath)
	err := goToolCmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Generated %s", output)
}
