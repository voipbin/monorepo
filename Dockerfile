FROM debian:stable-slim

ARG ASTERISK_GIT=https://github.com/asterisk/asterisk.git
ARG ASTERISK_VERSION=21.1.0
ARG ASTERISK_SOURCE_DIRECTORY=/asterisk
ARG CGSFUSE_VERSION=1.4.2

RUN apt-get update && apt-get upgrade

# Install utilities
RUN apt-get install -y \
    sngrep \
    sip-tester \
    htop \
    python3-apt \
    wget \
    ntp \
    psmisc \
    dnsutils \
    ngrep \
    vim \
    telnet \
    net-tools \
    curl \
    git \
    subversion \
    procps \
    iputils-ping \
    cron \
    systemctl

# Install required apt dependencies
RUN apt-get install -y \
    gcc \
    gdb \
    make \
    gzip \
    flex \
    bison \
    build-essential \
    autoconf \
    autotools-dev \
    automake \
    autogen \
    python3 \
    python3-dev \
    python3-pip \
    python3-mysqldb \
    python3-setuptools \
    uuid-dev \
    libcurl4-openssl-dev \
    libssl-dev \
    libncurses5-dev \
    libedit-dev \
    libxml2-dev \
    libsqlite3-dev \
    default-libmysqlclient-dev \
    xmlstarlet \
    binutils-dev \
    libsrtp2-dev \
    fuse

# Install gcsfuse
RUN curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v${CGSFUSE_VERSION}/gcsfuse_${CGSFUSE_VERSION}_amd64.deb
RUN dpkg --install gcsfuse_${CGSFUSE_VERSION}_amd64.deb

# Download Asterisk source
RUN git clone ${ASTERISK_GIT} ${ASTERISK_SOURCE_DIRECTORY}
COPY patches/ /tmp/patches

# Asterisk compilation & installation
WORKDIR ${ASTERISK_SOURCE_DIRECTORY}
RUN git checkout ${ASTERISK_VERSION}
RUN for i in /tmp/patches/*; do patch -p0 < $i; echo "patch applied: " $i > /var/log/asterisk_patch.log; done
RUN ./contrib/scripts/get_mp3_source.sh
RUN ./configure --with-jansson-bundled
RUN make menuselect.makeopts
RUN ./menuselect/menuselect --enable FORMAT_MP3 --enable DONT_OPTIMIZE --enable BETTER_BACKTRACES --enable CODEC_OPUS --enable RES_CONFIG_MYSQL --disable COMPILE_DOUBLE menuselect.makeopts
RUN make
RUN make install
