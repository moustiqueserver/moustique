# TLS/HTTPS Setup for Moustique

## Overview

Moustique supports TLS/HTTPS encryption for secure communication. This is **highly recommended** for production deployments, especially when the server is publicly accessible.

## Configuration

Enable TLS in your `config.yaml`:

```yaml
server:
  port: 33334  # Same port for both HTTP and HTTPS
  timeout: 30s
  allow_public: false
  tls:
    enabled: true  # Controls whether to use TLS on the port above
    cert_file: /path/to/certificate.pem
    key_file: /path/to/private-key.pem
```

**Important:** The `port` setting is used for both HTTP and HTTPS. The `tls.enabled` flag only controls whether TLS encryption is used on that port. You don't need separate ports for HTTP and HTTPS - just enable/disable TLS as needed.

## Creating TLS Certificates

### Option 1: Self-Signed Certificates (Development/Testing)

For development or internal use, you can create self-signed certificates:

```bash
# Generate private key
openssl genrsa -out server.key 2048

# Generate self-signed certificate (valid for 365 days)
openssl req -new -x509 -key server.key -out server.crt -days 365

# You'll be prompted for:
# - Country Name (2 letter code)
# - State or Province Name
# - Locality Name (city)
# - Organization Name
# - Organizational Unit Name
# - Common Name (use your server's hostname or IP)
# - Email Address
```

Then update your config:

```yaml
tls:
  enabled: true
  cert_file: /path/to/server.crt
  key_file: /path/to/server.key
```

**Note:** Clients connecting to a self-signed certificate will need to either:
- Add the certificate to their trusted CA store
- Disable certificate verification (not recommended for production)

### Option 2: Let's Encrypt (Production)

For production environments with a domain name, use Let's Encrypt for free, trusted certificates:

```bash
# Install certbot
sudo apt-get update
sudo apt-get install certbot

# Get certificate (HTTP-01 challenge)
sudo certbot certonly --standalone -d yourdomain.com

# Certificates will be created at:
# /etc/letsencrypt/live/yourdomain.com/fullchain.pem  (certificate)
# /etc/letsencrypt/live/yourdomain.com/privkey.pem    (private key)
```

Update your config:

```yaml
tls:
  enabled: true
  cert_file: /etc/letsencrypt/live/yourdomain.com/fullchain.pem
  key_file: /etc/letsencrypt/live/yourdomain.com/privkey.pem
```

**Important:**
- Let's Encrypt certificates expire after 90 days
- Set up automatic renewal with `certbot renew` in a cron job
- Restart Moustique after certificate renewal

#### Automatic Renewal

Add to crontab (`sudo crontab -e`):

```cron
# Renew certificates at 3am daily and restart Moustique if renewed
0 3 * * * certbot renew --quiet --deploy-hook "systemctl restart moustique"
```

### Option 3: Commercial Certificate Authority

If you have a certificate from a commercial CA (like DigiCert, GlobalSign, etc.):

1. You'll typically receive:
   - Your domain certificate (e.g., `yourdomain.com.crt`)
   - Intermediate certificate(s) (e.g., `intermediate.crt`)
   - Private key (e.g., `yourdomain.com.key`)

2. Combine the certificates:

```bash
cat yourdomain.com.crt intermediate.crt > fullchain.pem
```

3. Update config:

```yaml
tls:
  enabled: true
  cert_file: /path/to/fullchain.pem
  key_file: /path/to/yourdomain.com.key
```

## TLS Security Configuration

Moustique is configured with secure TLS settings:

- **Minimum TLS Version:** TLS 1.2
- **Cipher Suites:** Modern, secure ciphers only
  - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
  - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
  - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384

These settings provide:
- Forward secrecy (ECDHE)
- Strong encryption (AES-GCM)
- Protection against downgrade attacks

## File Permissions

Ensure proper permissions for certificate files:

```bash
# Private key should be readable only by the Moustique user
chmod 600 /path/to/private-key.pem
chown moustique:moustique /path/to/private-key.pem

# Certificate can be world-readable
chmod 644 /path/to/certificate.pem
chown moustique:moustique /path/to/certificate.pem
```

## Testing TLS Connection

### Using OpenSSL

```bash
# Test TLS connection
openssl s_client -connect localhost:33334 -showcerts

# Check certificate details
openssl s_client -connect localhost:33334 -showcerts < /dev/null 2>/dev/null | \
  openssl x509 -noout -text
```

### Using curl

```bash
# Test HTTPS connection
curl -v https://localhost:33334/version/running

# With self-signed certificate (skip verification)
curl -k https://localhost:33334/version/running
```

### Using Python Client

```python
from moustique import Moustique

# With valid certificate (use same port as configured in server)
client = Moustique(
    ip="yourdomain.com",
    port="33334",  # Use the same port as server's config.yaml
    client_name="test-client",
    username="user",
    password="pass",
    use_https=True
)

# With self-signed certificate (disable verification)
import requests
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

client = Moustique(
    ip="localhost",
    port="33334",
    client_name="test-client",
    username="user",
    password="pass",
    use_https=True,
    verify_ssl=False  # Only for self-signed certs in development
)
```

## Common Issues

### Error: "certificate signed by unknown authority"

**Cause:** Using self-signed certificate without adding it to trusted CAs.

**Solutions:**
1. Use a certificate from a trusted CA (Let's Encrypt, commercial CA)
2. Add the self-signed certificate to your system's trusted CA store
3. For development only: Disable certificate verification in clients

### Error: "TLS certificate file not found"

**Cause:** Certificate path in config is incorrect.

**Solution:** Verify the paths in your config.yaml point to existing files:

```bash
ls -la /path/to/cert.pem
ls -la /path/to/key.pem
```

### Error: "permission denied" when loading certificates

**Cause:** Moustique process doesn't have read permission on certificate files.

**Solution:** Fix file permissions (see File Permissions section above)

### Certificate Expiration

**Cause:** TLS certificates expire and need renewal.

**Solution:**
- For Let's Encrypt: Set up automatic renewal (see above)
- For commercial CAs: Monitor expiration dates and renew before expiry
- Set up monitoring/alerts for certificate expiration

```bash
# Check certificate expiration
openssl x509 -in /path/to/cert.pem -noout -enddate
```

## Ports

**Important:** Moustique uses the **same port** for both HTTP and HTTPS. The port is configured once in `config.yaml` under `server.port`, and the `tls.enabled` setting only controls whether TLS encryption is used on that port.

Example configurations:
- **HTTP on port 33334:** `port: 33334` with `tls.enabled: false`
- **HTTPS on port 33334:** `port: 33334` with `tls.enabled: true`

Standard web ports:
- **HTTP:** 80 (requires root or capability to bind to ports < 1024)
- **HTTPS:** 443 (requires root or capability to bind to ports < 1024)
- **Custom:** 33334 (default Moustique port, no special privileges required)

If running on standard ports (80 or 443), you'll need to either:

1. Run as root (not recommended)
2. Use `setcap` to allow binding to privileged ports:
   ```bash
   sudo setcap 'cap_net_bind_service=+ep' /path/to/moustique
   ```
3. Use a reverse proxy (nginx, Apache) on ports 80/443 forwarding to Moustique

## Reverse Proxy Setup (Recommended for Production)

For production, it's common to use nginx or Apache as a reverse proxy:

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers on;

    location / {
        proxy_pass http://localhost:33334;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

In this setup:
- Nginx handles TLS termination
- Moustique runs on port 33334 without TLS
- Clients connect to nginx on port 443 (HTTPS)

## Summary

**Development:**
```yaml
tls:
  enabled: true
  cert_file: ./certs/self-signed.crt
  key_file: ./certs/self-signed.key
```

**Production:**
```yaml
tls:
  enabled: true
  cert_file: /etc/letsencrypt/live/yourdomain.com/fullchain.pem
  key_file: /etc/letsencrypt/live/yourdomain.com/privkey.pem
```

**Production with Reverse Proxy:**
```yaml
tls:
  enabled: false  # nginx handles TLS
```
