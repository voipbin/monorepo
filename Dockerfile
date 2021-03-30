FROM debian:stable-slim

ARG ASTERISK_VERSION=18.3.0
ARG ASTERISK_SOURCE_DIRECTORY=/asterisk

RUN apt-get update

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
    iputils-ping

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
    libsrtp2-dev

# install packages for gcsfuse
RUN GCSFUSE_REPO=gcsfuse-`lsb_release -c -s` && echo "deb http://packages.cloud.google.com/apt $GCSFUSE_REPO main" > /etc/apt/sources.list.d/gcsfuse.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

# Install gcsfuse
RUN apt-get update
RUN apt-get install -y gcsfuse

# Download Asterisk source
RUN git clone --branch ${ASTERISK_VERSION} https://gerrit.asterisk.org/asterisk ${ASTERISK_SOURCE_DIRECTORY}

# Asterisk compilation & installation
WORKDIR ${ASTERISK_SOURCE_DIRECTORY}
RUN ./contrib/scripts/get_mp3_source.sh
RUN ./configure --with-jansson-bundled
RUN make menuselect.makeopts
RUN ./menuselect/menuselect --enable FORMAT_MP3 --enable DONT_OPTIMIZE --enable BETTER_BACKTRACES --enable CODEC_OPUS --enable RES_CONFIG_MYSQL --enable CDR_MYSQL --enable APP_MYSQL --disable COMPILE_DOUBLE --disable CHAN_SIP menuselect.makeopts
RUN make
RUN make install
