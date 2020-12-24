FROM debian:stable-slim

ARG ASTERISK_VERSION=18.1.1
ARG ASTERISK_SOURCE_DIRECTORY=/asterisk

RUN apt-get update

# Install required apt dependencies
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
    gcc \
    build-essential \
    curl \
    make \
    gzip \
    python3 \
    python3-dev \
    python3-pip \
    python3-mysqldb \
    python3-setuptools \
    libcurl4-openssl-dev \
    libssl-dev \
    flex \
    bison \
    libncurses5-dev \
    uuid-dev \
    libedit-dev \
    libxml2-dev \
    libsqlite3-dev \
    git \
    xmlstarlet \
    binutils-dev \
    autoconf \
    autotools-dev \
    automake \
    autogen \
    gdb \
    subversion \
    libsrtp2-dev \
    unixodbc \
    unixodbc-dev \
    curl \
    procps
    # odbc-mariadb

# Install Asterisk pip dependencies
RUN pip3 install alembic

# install packages for gcsfuse
RUN GCSFUSE_REPO=gcsfuse-`lsb_release -c -s` && echo "deb http://packages.cloud.google.com/apt $GCSFUSE_REPO main" > /etc/apt/sources.list.d/gcsfuse.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

# Install gcsfuse
RUN apt-get update
RUN apt-get install -y gcsfuse

# Copy service accounts
COPY etc/service_accounts /service_accounts

# Download Asterisk source
RUN mkdir ${ASTERISK_SOURCE_DIRECTORY}
RUN curl -s http://downloads.asterisk.org/pub/telephony/asterisk/asterisk-${ASTERISK_VERSION}.tar.gz | tar xz -C ${ASTERISK_SOURCE_DIRECTORY} --strip-components=1

# Asterisk compilation & installation
WORKDIR ${ASTERISK_SOURCE_DIRECTORY}
RUN ./contrib/scripts/get_mp3_source.sh
RUN ./configure --with-jansson-bundled
RUN make menuselect.makeopts
RUN ./menuselect/menuselect --enable FORMAT_MP3 --enable DONT_OPTIMIZE --enable BETTER_BACKTRACES --enable CODEC_OPUS --enable RES_CONFIG_ODBC --enable RES_ODBC --disable COMPILE_DOUBLE --disable CHAN_SIP menuselect.makeopts
RUN make
RUN make install

# Copy configuration files
COPY etc/asterisk /etc/asterisk

# Copy scripts
COPY etc/scripts/* /

# Start
CMD ["/bin/bash", "/start.sh"]
