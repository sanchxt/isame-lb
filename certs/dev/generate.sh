#!/bin/bash

# Script to generate self-signed TLS certificates for development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Generating self-signed TLS certificates for development..."
echo

# Generate private key
echo "1. Generating RSA private key (server.key)..."
openssl genrsa -out server.key 2048

# Generate certificate signing request (CSR)
echo "2. Generating certificate signing request..."
openssl req -new -key server.key -out server.csr \
  -subj "/C=US/ST=Development/L=Local/O=Isame-LB-Dev/CN=localhost"

# Generate self-signed certificate valid for 365 days
echo "3. Generating self-signed certificate (server.crt)..."
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt \
  -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")

# Clean up CSR
rm server.csr

# Set appropriate permissions
chmod 600 server.key
chmod 644 server.crt

echo
echo "✓ Development certificates generated successfully!"
echo
echo "Files created:"
echo "  - server.key: Private key (keep this secure!)"
echo "  - server.crt: Self-signed certificate"
echo
echo "To use these certificates:"
echo "  1. Update your config file with:"
echo "     tls:"
echo "       enabled: true"
echo "       cert_file: \"certs/dev/server.crt\""
echo "       key_file: \"certs/dev/server.key\""
echo
echo "  2. Start the load balancer with HTTPS support"
echo
echo "  3. Test with: curl -k https://localhost:8443/health"
echo "     (use -k flag to allow self-signed cert)"
