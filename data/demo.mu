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

		before <<SCRIPT
		console.log(`start running ${requestName}`)
		SCRIPT

		after <<SCRIPT
		console.log(`done running ${requestName} with status ${responseStatus} (${requestDuration} sec)`)
		SCRIPT
	}

	get colors {
		url '/colors/'
	}
}
