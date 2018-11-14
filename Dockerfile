FROM ubuntu:xenial

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
      curl \
      iptables \
      iproute2 \
      iputils-ping \
      dnsutils \
      netcat \
      tcpdump \
      net-tools \
      libc6-dbg gdb \
      elvis-tiny \
      lsof \
      linux-tools-generic \
      sudo &&  apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir /run/url-cache

ADD url-lookup /usr/local/bin/url-lookup

ENTRYPOINT ["/usr/local/bin/url-lookup"]
