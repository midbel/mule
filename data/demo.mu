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

	beforeEach <<SCRIPT
	console.log(`start running ${requestName}: ${mule.request.url}`)
	SCRIPT

	afterEach <<SCRIPT
	console.log(`done running ${requestName} with status ${responseStatus} (${requestDuration} sec)`)
	console.log(mule.response.headers)
	SCRIPT

	url http://localhost:9001

	get animals {
	  url '/animals/'
	}

	get colors {
		url '/colors/'
	}
}
