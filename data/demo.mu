headers {
	accept          application/json
	content-type    application/json
}

collection basic {
	tls {
	  certFile 'tmp/tls/demo/cert.pem'
	  certKey  'tmp/tls/demo/key.pem'
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
