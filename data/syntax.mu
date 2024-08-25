collection name {
	tls {
		certFile
		certKey
		certCA
		serverName
		insecure
		minVersion
		maxVersion
	}

	variables {
		var1 foo
		var2 bar
	}

	url http://localhost

	auth bearer "abc.defgh.xyz"
	auth bearer {
		token "abc.defgh.xyz"
	}
	auth basic {
		username foobar
		password tmp123
	}

	headers {
		accept "application/json"
		accept-encoding gz
	}
	query {
		order time
		limit 100
		page  1
	}

	before <<SCRIPT
	SCRIPT

	after <<SCRIPT
	SCRIPT

	beforeEach <<SCRIPT
	SCRIPT

	afterEach <<SCRIPT
	SCRIPT

	get request {
		depends r1 r2
		expect  200
		url http://localhost

		retry 100
		timeout 100

		tls {
			certFile
			certKey
			certCA
			serverName
			insecure
			minVersion
			maxVersion
		}

		cookie {

		}

		body @readfile `body.json ${var}`

		headers {
			accept "application/json"
			accept-encoding gz lz
		}
		query {
			order time
			limit 100
			page  1
		}

		before <<SCRIPT
		SCRIPT

		after <<SCRIPT
		SCRIPT
	}
}