# Cert

This directory contains a self-signed certificate for
localhost so that stripe-mock can run with HTTPS. You can
generate a new one with this command:

    openssl req -x509 -newkey rsa:4096 -keyout cert/key.pem -out cert/cert.pem -days 3650 -nodes -subj '/CN=localhost'

And because certificates are bundled with the executable,
you'll need to put the new certificate into a `*.go` file
with go-bindata:

    go generate
