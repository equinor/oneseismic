openssl req -x509 -newkey rsa:4096 -keyout key.pem -out az.pem -days 365 -subj '/CN=az' --nodes -addext "subjectAltName = DNS:az"
