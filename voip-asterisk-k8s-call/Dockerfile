FROM $ASTERISK-IMAGE

# Copy service accounts
COPY voip-asterisk-k8s-call/etc/service_accounts /service_accounts

# Copy configuration files
COPY voip-asterisk-k8s-call/etc/asterisk /etc/asterisk

# Copy scripts
COPY voip-asterisk-k8s-call/etc/scripts/* /

# Download asterisk-exporter
RUN ["wget", "https://github.com/pchero/asterisk-exporter/releases/download/0.0.4/asterisk-exporter-0.0.4-linux-amd64", "-O", "/asterisk-exporter"]
RUN ["chmod", "755", "/asterisk-exporter"]
