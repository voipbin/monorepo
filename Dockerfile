FROM registry.gitlab.com/voipbin/voip/asterisk-docker:20.3.0

# Copy service accounts
COPY etc/service_accounts /service_accounts

# Copy configuration files
COPY etc/asterisk /etc/asterisk

# Copy scripts
COPY etc/scripts/* /
