# go-dumpotron
Take pprof dumps when a go process eats too much ram or cpu.

This has been running on `prometheus.locotorp.info since Jan 2020` 

## Requirements
- graphviz
- go 1.13
- ipfs-cluster-ctl

## Running
The following ENVs need to be set:
- PPROF_AUTH_PASS
- IPFS_CLUSTER_AUTH
- GITHUB_TOKEN

### Build
```
go build
```
Docker:
```
docker build -t go-dumpotron .
```

### Run
Get credentials from the 1Password note named [go-dumpotron credentials](https://start.1password.com/open/i?a=4XNRW7JPXZEI7C7CEIAF27VTSQ&h=protocollabs.1password.com&i=5fqwpic7gp2pxu3oc3i4elgknm&v=hgaw43xamtkvt35xfx3gywnppa) in the `Infra team` vault

#### As daemon, HTTP server accepting Alertmanager webhook calls
```
# Prepend `LOG_LEVEL=debug` for debugging logs
source .env && ./go-dumpotron -daemon
```

Docker:
```
docker run --rm --net=host --name=go-dumpotron --env-file=.dockerenv ipfsshipyardbot/go-dumpotron
```

#### One-time, generate pprof archive locally for specific instance
```
# Prepend `LOG_LEVEL=debug` for debugging logs
source .env && ./go-dumpotron gateway-bank1-ewr1.dwebops.net
# or by passing in a basic auth passwd for the endpoint
PPROF_AUTH_PASS=THE_ADMIN_HTTPASSWD ./go-dumpotron gateway-bank1-ewr1.dwebops.net
```

Docker:
```
mkdir /tmp/dumps
docker run -it --env-file=.dockerenv -w /tmp/pprofs -v $(pwd):/tmp/pprofs ipfsshipyardbot/go-dumpotron gateway-bank1-ewr1.dwebops.net
```

### Webhook Test
```
curl localhost:9096 -d @fixtures/sample.json
```
