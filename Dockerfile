FROM golang:alpine3.11
MAINTAINER George Masgras <george.magras@protocol.ai>

RUN apk update && apk add curl git ca-certificates graphviz make g++

# Get su-exec, a very minimal tool for dropping privileges,
# and tini, a very minimal init daemon for containers
ENV SUEXEC_VERSION v0.2
RUN set -x \
  && cd /tmp \
  && git clone https://github.com/ncopa/su-exec.git \
  && cd su-exec \
  && git checkout -q $SUEXEC_VERSION \
  && make

# Compile ipfs-cluster-ctl
ENV IPFS_CLUSTER_VERSION v0.12.1
RUN set -x \
  && cd /tmp \
  && git clone https://github.com/ipfs/ipfs-cluster.git \
  && cd ipfs-cluster \
  && git checkout -q $IPFS_CLUSTER_VERSION \
  && go build -o /tmp/ipfs-cluster-ctl ./cmd/ipfs-cluster-ctl/*.go

ENV SRC_DIR /go-dumpotron
# Download packages first so they can be cached.
COPY go.mod go.sum $SRC_DIR/
RUN cd $SRC_DIR \
  && go mod download

COPY . $SRC_DIR

# Build the thing.
# Also: fix getting HEAD commit hash via git rev-parse.
RUN cd $SRC_DIR \
  && go build


# Now comes the actual target image, which aims to be as small as possible.
FROM golang:alpine3.11
#MAINTAINER George Masgras <george.magras@protocol.ai>

# Get the TLS CA certificates, they're not provided by busybox.
RUN apk update && apk add ca-certificates graphviz

  ## Get the binary from the build container.
ENV SRC_DIR /go-dumpotron
COPY --from=0 $SRC_DIR/go-dumpotron /usr/local/bin/go-dumpotron
COPY --from=0 /tmp/su-exec/su-exec /sbin/su-exec
COPY --from=0 /tmp/ipfs-cluster-ctl /sbin/ipfs-cluster-ctl

# add user
RUN  adduser -D -H -u 1000 dumpotron \
  && chown dumpotron:users /usr/local/bin/go-dumpotron

# webhook TCP should be exposed to the public
EXPOSE 9096

ENTRYPOINT ["su-exec", "dumpotron", "go-dumpotron"]

CMD ["-daemon"]
