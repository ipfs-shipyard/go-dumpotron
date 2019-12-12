package main

import (
	"webhook-prometheus"
	"testing"
)

func TestCollect(t *testing.T) {
	pprofs := main.NewPprofRequest("gateway-bank2-ams1.dwebops.net")
	pprofs.Collect()
}
