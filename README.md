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

### Build:
```
go build
```
Docker:
```
docker build -t go-dumpotron .
```

### Run
#### As daemon, HTTP server accepting Alertmanager webhook calls
```
# Prepend `LOG_LEVEL=debug` for debugging logs
./go-dumpotron -daemon
```

Docker:
```
docker run --rm --net=host --name=go-dumpotron --env-file=.dockerenv go-dumpotron
```

#### One-time, generate pprof archive locally for specific instance
```
# Prepend `LOG_LEVEL=debug` for debugging logs
./go-dumpotron gateway-bank1-ewr1.dwebops.net
```

Docker:
```
mkdir /tmp/dumps
docker run --rm --net=host --name=go-dumpotron --env-file=.dockerenv -v /tmp/dumps:/tmp go-dumpotron gateway-bank1-ewr1.dwebops.net
```


### Webhook Test:
```
curl localhost:9096 -d @fixtures/sample.json
```
