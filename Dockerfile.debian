FROM golang:1.13.4-buster
MAINTAINER George Masgras <george.magras@protocol.ai>

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
FROM golang:1.13.4-buster
#MAINTAINER George Masgras <george.magras@protocol.ai>

# Get the TLS CA certificates, they're not provided by busybox.
RUN apt-get update && apt-get install -y ca-certificates graphviz

# Get su-exec, a very minimal tool for dropping privileges,
# and tini, a very minimal init daemon for containers
ENV SUEXEC_VERSION v0.2
RUN set -x \
  && cd /tmp \
  && git clone https://github.com/ncopa/su-exec.git \
  && cd su-exec \
  && git checkout -q $SUEXEC_VERSION \
  && make \
  && cp /tmp/su-exec/su-exec /sbin/su-exec

# Install ipfs-cluster-ctl
RUN set -x \
  && cd /tmp \
  && curl -O  https://dist.ipfs.io/ipfs-cluster-ctl/v0.12.1/ipfs-cluster-ctl_v0.12.1_linux-amd64.tar.gz \
  && tar zxf ipfs-cluster-ctl*.tar.gz \
  && cp /tmp/ipfs-cluster-ctl/ipfs-cluster-ctl /usr/local/bin/ipfs-cluster-ctl

  ## Get the binary from the build container.
ENV SRC_DIR /go-dumpotron
COPY --from=0 $SRC_DIR/go-dumpotron /usr/local/bin/go-dumpotron

# add user
RUN  useradd --system --no-create-home --no-user-group --uid 1000 dumpotron \
  && chown dumpotron:users /usr/local/bin/go-dumpotron

# webhook TCP should be exposed to the public
EXPOSE 9096

CMD ["su-exec", "dumpotron", "go-dumpotron"]
