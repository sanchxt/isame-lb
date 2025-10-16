#!/bin/bash

# Script to generate test certificates for TLS testing
# These are self-signed certificates for testing purposes only

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Generating test certificates..."

# Generate valid certificate and key
echo "1. Generating valid certificate (server.crt, server.key)..."
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout server.key \
  -out server.crt \
  -days 365 \
  -subj "/C=US/ST=Test/L=Test/O=Test/CN=localhost"

# Generate expired certificate
echo "2. Generating expired certificate (expired.crt, expired.key)..."
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout expired.key \
  -out expired.crt \
  -days 1 \
  -subj "/C=US/ST=Test/L=Test/O=Test/CN=expired.localhost"

# Backdate the expired certificate to make it actually expired
# Create a certificate that expired yesterday
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout expired.key \
  -out expired.crt \
  -days -1 \
  -subj "/C=US/ST=Test/L=Test/O=Test/CN=expired.localhost" 2>/dev/null || {
    # If that doesn't work, use faketime or create a properly expired cert
    echo "Creating expired cert using date manipulation..."
    faketime 'last year' openssl req -x509 -newkey rsa:2048 -nodes \
      -keyout expired.key \
      -out expired.crt \
      -days 1 \
      -subj "/C=US/ST=Test/L=Test/O=Test/CN=expired.localhost" 2>/dev/null || {
        echo "Warning: Could not create truly expired certificate. Using valid cert as placeholder."
        cp server.crt expired.crt
        cp server.key expired.key
      }
}

# Generate mismatched key pair (certificate with different key)
echo "3. Generating mismatched key pair (mismatched.crt with wrong key)..."
# Generate first cert
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout temp.key \
  -out mismatched.crt \
  -days 365 \
  -subj "/C=US/ST=Test/L=Test/O=Test/CN=mismatched.localhost"

# Generate different key
openssl genrsa -out mismatched.key 2048 2>/dev/null
rm temp.key

# Create invalid certificate file
echo "4. Creating invalid certificate (invalid.crt)..."
echo "-----BEGIN CERTIFICATE-----" > invalid.crt
echo "This is not a valid certificate content" >> invalid.crt
echo "Just some random text that will fail parsing" >> invalid.crt
echo "-----END CERTIFICATE-----" >> invalid.crt
echo "-----BEGIN PRIVATE KEY-----" > invalid.key
echo "This is not a valid private key" >> invalid.key
echo "-----END PRIVATE KEY-----" >> invalid.key

echo ""
echo "Test certificates generated successfully!"
echo "Files created:"
echo "  - server.crt/server.key (valid certificate)"
echo "  - expired.crt/expired.key (expired certificate)"
echo "  - mismatched.crt/mismatched.key (mismatched key pair)"
echo "  - invalid.crt/invalid.key (malformed certificate)"
