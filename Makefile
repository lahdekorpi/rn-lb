#!/bin/bash
# install.sh - Install RN-LB systemd services

set -e

MODE=${1:-monolithic}  # monolithic or split

# Create user and group
if ! id rn-lb >/dev/null 2>&1; then
    useradd -r -s /bin/false -d /var/lib/rn-lb rn-lb
fi

if [ "$MODE" = "split" ]; then
    if ! id rn-lb-proxy >/dev/null 2>&1; then
        useradd -r -s /bin/false -d /var/lib/rn-lb-proxy rn-lb-proxy
    fi
fi

# Create directories
mkdir -p /etc/rn-lb
mkdir -p /var/lib/rn-lb
mkdir -p /var/log/rn-lb

if [ "$MODE" = "split" ]; then
    mkdir -p /var/log/rn-lb-proxy
    chown rn-lb-proxy:rn-lb-proxy /var/log/rn-lb-proxy
fi

# Set permissions
chown rn-lb:rn-lb /var/lib/rn-lb
chown rn-lb:rn-lb /var/log/rn-lb
chmod 750 /etc/rn-lb
chmod 750 /var/lib/rn-lb

# Install binaries
make install

# Install systemd services
if [ "$MODE" = "monolithic" ]; then
    install -m 644 systemd/rn-lb.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable rn-lb.service
    echo "Installed rn-lb.service"
    echo "Edit /etc/rn-lb/config.yaml and then: systemctl start rn-lb"
else
    install -m 644 systemd/rn-lb-main.service /etc/systemd/system/
    install -m 644 systemd/rn-lb-proxy.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable rn-lb-proxy.service
    systemctl enable rn-lb-main.service
    echo "Installed rn-lb-main.service and rn-lb-proxy.service"
    echo "Edit /etc/rn-lb/config.yaml and /etc/rn-lb/proxy-config.yaml"
    echo "Then: systemctl start rn-lb-proxy && systemctl start rn-lb-main"
fi

echo "Installation complete!"