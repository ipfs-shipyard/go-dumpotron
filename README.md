# go-dumpotron
Take pprof dumps when a go process eats too much ram or cpu.

!!! THIS IS NOT IN ANY WAY READY/SUPPORTED YET !!!

## Requirements
- graphviz
- go 1.13
- ipfs-cluster-ctl

## Running
The following ENVs need to be set:
- PPROF_AUTH_PASS
- IPFS_CLUSTER_AUTH
- GITHUB_TOKEN

Build:
```
go build
```

Run:
```
./go-dumpotron
```

Test:
```
curl localhost:9096 -d @fixtures/sample.json
```
