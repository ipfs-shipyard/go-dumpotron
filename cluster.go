package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"bytes"
	"log"
)

var ipfsClusterClient *IPFSCluster

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

func setupIPFSClusterClient(basicAuth string) {
	ipfsClusterClient = &IPFSCluster{
		host: "/dnsaddr/cluster.ipfs.io",
		basicAuth: basicAuth }
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

func (c *IPFSCluster) AddAndPin(archivePath string) (string, error) {
	cids, err := ipfsClusterClient.Add(archivePath)
	if err != nil {
		return "", fmt.Errorf("Add to cluster %s: %v", archivePath, err)
	}

	log.Printf("added cids: %v", cids)

	err = ipfsClusterClient.Pin(cids)
	if err != nil {
		return "", fmt.Errorf("Pin to cluster %s: %v", cids, err)
	}
	dirCid := cids[len(cids) -1]

	// https://ipfs.io/ipfs/CID/archive.tar.gz
	return fmt.Sprintf("https://ipfs.io/ipfs/%s/%s", dirCid, path.Base(archivePath)),nil
	// return fmt.Sprintf("Pinned %s", cids), nil
}
