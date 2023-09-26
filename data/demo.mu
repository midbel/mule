headers {
	accept          application/json
	content-type    application/json
}

collection basic {
	tls {
	  certFile 'tmp/tls/client1/cert.pem'
	  certKey  'tmp/tls/client1/key.pem'
		certCA   'tmp/tls/ca/'
		insecure true
	}

	url https://localhost:9001

	get animals {
	  url '/animals/'
	}

	get colors {
		url '/colors/'
	}
}
