cert.pem: cert.csr ca.pem ca-key.pem
	openssl x509 -req -days 3650 -in cert.csr -CA ca.pem -CAkey ca-key.pem -out $@ -set_serial 1

cert.csr: cert-key.pem
	openssl req -new -sha256 -key $< -out $@ -subj '/CN=otaru-server/O=Test/C=JP'

cert-key.pem:
	openssl ecparam -out $@ -name prime256v1 -genkey


ca.pem: ca-key.pem
	openssl req -key $< -new -x509 -days 3650 -sha256 -out $@ -extensions v3_ca --subj '/C=JP'

ca-key.pem:
	openssl ecparam -out $@ -name prime256v1 -genkey



