FROM registry.gitlab.com/voipbin/voip/asterisk-docker:19.4.1

# Copy service accounts
COPY etc/service_accounts /service_accounts

# Copy configuration files
COPY etc/asterisk /etc/asterisk

# Copy scripts
COPY etc/scripts/* /

# Download asterisk-exporter
RUN ["wget", "https://github.com/pchero/asterisk-exporter/releases/download/0.0.2/asterisk-exporter-0.0.2-linux-amd64", "-O", "/asterisk-exporter"]
RUN ["chmod", "755", "/asterisk-exporter"]
