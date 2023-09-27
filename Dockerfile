FROM registry.gitlab.com/voipbin/voip/asterisk-docker:5865060c5540007f3d00d13338d6dbc197f03700

# Copy service accounts
COPY etc/service_accounts /service_accounts

# Copy configuration files
COPY etc/asterisk /etc/asterisk

# Copy scripts
COPY etc/scripts/* /
