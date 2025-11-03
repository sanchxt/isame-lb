# TLS Certificates

This directory contains TLS certificates for the Isame Load Balancer.

## Development Certificates

For local development and testing, use the self-signed certificates in the `dev/` directory.

### Generating Development Certificates

```bash
cd certs/dev
./generate.sh
```

This will create:

- `server.crt`: Self-signed certificate valid for 365 days
- `server.key`: RSA 2048-bit private key

### Using Development Certificates

Update your configuration file (`configs/dev.yaml` or similar):

```yaml
tls:
  enabled: true
  cert_file: "certs/dev/server.crt"
  key_file: "certs/dev/server.key"
  min_version: "1.2" # Optional: TLS 1.2 or 1.3
```

Then start the load balancer:

```bash
./isame-lb --config configs/dev.yaml
```

Test HTTPS endpoint:

```bash
# Using curl (with -k to allow self-signed cert)
curl -k https://localhost:8443/health

# Check certificate details
openssl s_client -connect localhost:8443 -showcerts
```
