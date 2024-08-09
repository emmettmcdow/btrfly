# TLS for server endpoints
For now, we need to generate self-signed certificates to test this with HTTPS.
Just run the following commands in this directory:
```
openssl ecparam -genkey -name secp384r1 -out server.key
openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650

```
