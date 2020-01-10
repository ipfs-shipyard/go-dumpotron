package main

import (
	"bytes"
	"net/http"
	"encoding/json"
	"time"
	"io/ioutil"
	"log"
	"fmt"
	"path"
	"os"
	"os/exec"
)

const GatewaysDomain = ".dwebops.net"

type PprofRequest struct {
	Instance string
	netClient *http.Client
	dumpDir string
	tempDir string
	profiles []Profile
	ipfsVersion IPFSVersion
	httpasswd string
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

func NewPprofRequest(instance string, httpasswd string) (*PprofRequest, error) {
	netClient := &http.Client{
		Timeout: time.Second * 120,
	}

	ipfsVersion, err := fetchVersion(instance, netClient)
	if err != nil {
		return &PprofRequest{}, fmt.Errorf("%s: Failed to fetch go-ipfs version: %v", instance, err)
	}
	log.Printf("Instance %s running version: %s-%s", instance, ipfsVersion.Version, ipfsVersion.Commit)


	tempDir, err := ioutil.TempDir("", "pprof")
	if err != nil {
		return &PprofRequest{}, fmt.Errorf("Failed to create tempDir: %v", err)
	}
	t := time.Now()
	dumpDir := fmt.Sprintf("%s/%s_%s-%s_%s", tempDir, instance, ipfsVersion.Version, ipfsVersion.Commit, t.Format(time.RFC3339))
	err = os.Mkdir(dumpDir, 0700)
	if err != nil {
		return &PprofRequest{}, fmt.Errorf("Failed to create dumpDir: %v", err)
	}
	// DEBUG
	log.Printf("dumpDir: %v", dumpDir)

	profiles := []Profile{{url: "/debug/pprof/goroutine?debug=2"}}
	request := PprofRequest{
		Instance: instance + GatewaysDomain,
		netClient: netClient,
		dumpDir: dumpDir,
		tempDir: tempDir,
		profiles: profiles,
		httpasswd: httpasswd,
		ipfsVersion: ipfsVersion}

	return &request, nil
}

//
// func (r *PprofRequest) Collect() {
// 	for _, req
// }

func (r *PprofRequest) Collect() (string, error) {
	log.Printf("Collecting pprofs for %s to %s", r.Instance, r.dumpDir)
	err := r.goroutineStacks()
	if err != nil { return "", fmt.Errorf("%s: fetch goroutine stacks: %v", r.Instance, err) }

	err = r.goroutineProfile()
	if err != nil { return "", fmt.Errorf("%s: fetch goroutine profile: %v", r.Instance, err) }

	err = r.heapProfile()
	if err != nil { return "", fmt.Errorf("%s: fetch heap profile: %v", r.Instance, err) }

	err = r.cpuProfile()
	if err != nil { return "", fmt.Errorf("%s: fetch CPU profile: %v", r.Instance, err) }

	err = r.mutexProfile()
	if err != nil { return "", fmt.Errorf("%s: fetch mutex profile: %v", r.Instance, err) }

	archivePath, err := r.createArchive()
	if err != nil { return "", fmt.Errorf("%s: create archive: %v", r.Instance, err) }

	//return archivePath, nil
	cidUrl, err := r.addAndPinToCluster(archivePath)
	if err != nil { return "", fmt.Errorf("%s: add to cluster: %s: %v", r.Instance, archivePath, err) }

	log.Printf("URL to pinned archive: %s", cidUrl)
	return cidUrl, nil
}

func (r *PprofRequest) goroutineStacks() (error) {
	_, err := r.fetchPprof("/debug/pprof/goroutine?debug=2", "goroutine.stacks")
	if err != nil { return err }
	//DEBUG
	//log.Println(profile)
	return nil
}

func (r *PprofRequest) goroutineProfile() (error) {
	profile, err := r. fetchPprof("/debug/pprof/goroutine", "goroutine.pprof")
	if err != nil { return err }
	//DEBUG
	//log.Println(profile)
	err = generateSVG(profile)
	if err != nil { return err }
	return nil
}

func (r *PprofRequest) heapProfile() (error) {
	profile, err := r.fetchPprof("/debug/pprof/heap", "heap.pprof")
	if err != nil { return err }
	//DEBUG
	// log.Println(profile)
	err = generateSVG(profile)
	if err != nil { return err }
	return nil
}

// Collecting cpu profile (~30s)
func (r *PprofRequest) cpuProfile() (error) {
	profile, err := r.fetchPprof("/debug/pprof/profile", "cpuProfile.pprof")
	if err != nil { return err }
	//DEBUG
	// log.Println(profile)
	err = generateSVG(profile)
	if err != nil { return err }
	return nil
}

func (r *PprofRequest) mutexProfile() (error) {
	// Enabling mutex profiling"
	postURLEnable := fmt.Sprintf("https://%s%s", r.Instance, "/debug/pprof-mutex/?fraction=4")
	postReqEnable, err := http.NewRequest("POST", postURLEnable, nil)
	postReqEnable.SetBasicAuth("admin", r.httpasswd)
	//DEBUG
	// log.Println(postReqEnable)
	resp, err := r.netClient.Do(postReqEnable)
	if err != nil { return err }
	//DEBUG
	log.Println(resp)

	// Waiting for mutex data to be updated
	time.Sleep(30 * time.Second)

	debug, err := r.fetchPprof("/debug/pprof/mutex?debug=2", "mutexDebug.txt")
	if err != nil { return err }
	log.Println(debug)

	profile, err := r.fetchPprof("/debug/pprof/mutex", "mutexProfile.pprof")
	if err != nil { return err }
	log.Println(profile)

	// Disabling mutex profiling"
	postURLDisable := fmt.Sprintf("https://%s%s", r.Instance, "/debug/pprof-mutex/?fraction=0")
	postReqDisable, err := http.NewRequest("POST", postURLDisable, nil)
	postReqDisable.SetBasicAuth("admin", r.httpasswd)
	//DEBUG
	// log.Println(postReqDisable)
	resp, err = r.netClient.Do(postReqDisable)
	if err != nil { return err }
	// DEBUG
	// log.Println(resp)

	err = generateSVG(profile)
	if err != nil { return err }
	return nil
}

func (r *PprofRequest) fetchPprof(location string, localFilename string) (string, error) {
	url := fmt.Sprintf("https://%s%s", r.Instance, location)
	fileLocation := r.dumpDir + "/" + localFilename
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("admin", r.httpasswd)

	log.Printf("fetching %s profile from %s", localFilename, url)
	resp, err := r.netClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	if resp.StatusCode > 400 {
		return "", fmt.Errorf("HTTP request for %s failed with: %d", location, resp.StatusCode)
	}

	// log.Printf("resp: %v", resp)
	body, err := ioutil.ReadAll(resp.Body)
	err = ioutil.WriteFile(fileLocation, body, 0644)
	if err != nil {
		return "", err
	}

	return fileLocation, nil
}

func fetchVersion(instance string, netClient *http.Client) (IPFSVersion, error) {
	var ipfsVersion IPFSVersion
	url := fmt.Sprintf("http://%s%s%s", instance , GatewaysDomain, "/api/v0/version")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ipfsVersion,fmt.Errorf("failed to fetch version: %v", err)
	}

	resp, err := netClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return ipfsVersion,fmt.Errorf("failed to fetch version: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	//DEBUG
	log.Printf("fetched version: %s", body)
	err = json.Unmarshal(body, &ipfsVersion)
	if err != nil {
		return ipfsVersion, fmt.Errorf("failed to unmarshal version from JSON: %v", err)
	}

	return ipfsVersion, nil
}

func (r *PprofRequest) createArchive() (string, error) {
	archivePath := fmt.Sprintf("%s/%s.tar.gz", r.tempDir, path.Base(r.dumpDir))
	//DEBUG
	log.Printf("creating archive: %s", archivePath)
	tarCmd:= exec.Command("tar", "czf", archivePath, "-C", r.tempDir,  path.Base(r.dumpDir))

	stderr, err := tarCmd.StderrPipe()
	if err != nil {
		return archivePath, fmt.Errorf("create archive: %s: %v", archivePath , err)
	}

	if err := tarCmd.Start(); err != nil {
		return archivePath, fmt.Errorf("create archive: %s: %v", archivePath , err)
	}

	result := new(bytes.Buffer)
	result.ReadFrom(stderr)

	if err := tarCmd.Wait(); err != nil {
		log.Printf("Result of creating %s: %s", archivePath, result)
		return archivePath, fmt.Errorf("create archive: %s: %v", archivePath , err)
	}

	log.Printf("Generated %s", archivePath)
	return archivePath, nil
}


func generateSVG(profilePath string) (error) {
	svgOutput := profilePath + ".svg"
	goToolCmd:= exec.Command("go", "tool", "pprof", "-symbolize=remote", "-svg", "-output", svgOutput, profilePath)
	err := goToolCmd.Run()
	if err != nil { return err }
	log.Printf("Generated %s", svgOutput)
	return nil
}

func (r *PprofRequest) addAndPinToCluster(archivePath string) (string, error) {
	cluster := NewIPFSCluster()
	cids, err := cluster.Add(archivePath)
	if err != nil {
		return "", fmt.Errorf("Add to cluster %s: %v", archivePath, err)
	}

	log.Printf("added cids: %v", cids)

	err = cluster.Pin(cids)
	if err != nil {
		return "", fmt.Errorf("Pin to cluster %s: %v", cids, err)
	}
	dirCid := cids[len(cids) -1]

	// https://ipfs.io/ipfs/CID/archive.tar.gz
	return fmt.Sprintf("https://ipfs.io/ipfs/%s/%s", dirCid, path.Base(archivePath)),nil
	// return fmt.Sprintf("Pinned %s", cids), nil
}
