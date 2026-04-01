#!/bin/bash
set -euo pipefail

echo "Installing ${app_name} version ${app_version}"

apt-get update
apt-get install -y nginx

mkdir -p /opt/webapp
cat > /opt/webapp/config.json <<EOF
{
  "name": "${app_name}",
  "version": "${app_version}",
  "installed_at": "$(date -Iseconds)"
}
EOF

cat > /var/www/html/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
  <title>${app_name}</title>
</head>
<body>
  <h1>${app_name} v${app_version}</h1>
  <p>Deployed via Packer + Cloud-Init</p>
</body>
</html>
EOF

systemctl enable nginx
systemctl restart nginx

echo "Installation complete"
