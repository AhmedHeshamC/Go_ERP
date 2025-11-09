# SSL Certificate Setup for ERPGo Production

This directory contains SSL certificates for the ERPGo production deployment.

## Files Required

1. **cert.pem** - Your SSL certificate (full chain)
2. **key.pem** - Your private key
3. **chain.pem** - Certificate chain file (intermediate certificates)

## Getting SSL Certificates

### Option 1: Let's Encrypt (Recommended for production)

```bash
# Install certbot
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx

# Generate certificates
sudo certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com

# Copy certificates to this directory
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem configs/nginx/ssl/cert.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem configs/nginx/ssl/key.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/chain.pem configs/nginx/ssl/chain.pem
```

### Option 2: Self-signed certificates (for testing)

```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout configs/nginx/ssl/key.pem \
    -out configs/nginx/ssl/cert.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=yourdomain.com"

# Create chain file (copy cert for self-signed)
cp configs/nginx/ssl/cert.pem configs/nginx/ssl/chain.pem
```

### Option 3: Commercial certificates

1. Purchase SSL certificate from a CA
2. Generate CSR:
   ```bash
   openssl req -new -newkey rsa:2048 -nodes \
       -keyout configs/nginx/ssl/key.pem \
       -out configs/nginx/ssl/cert.csr \
       -subj "/C=US/ST=State/L=City/O=Organization/CN=yourdomain.com"
   ```
3. Submit CSR to your CA
4. Download certificates and place them in this directory

## Security Notes

- Set proper permissions:
  ```bash
  chmod 600 configs/nginx/ssl/key.pem
  chmod 644 configs/nginx/ssl/cert.pem
  chmod 644 configs/nginx/ssl/chain.pem
  ```
- Never commit private keys to version control
- Use strong cryptographic parameters (RSA 2048+ or ECC)
- Enable automatic renewal for Let's Encrypt certificates

## Certificate Renewal (Let's Encrypt)

Add to crontab for automatic renewal:
```bash
0 3 * * * certbot renew --quiet --deploy-hook "docker-compose -f docker-compose.prod.yml restart nginx"
```