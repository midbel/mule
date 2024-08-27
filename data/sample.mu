# this is a comment
# this is another comment

url http://localhost:3000

variables {
	token abcdef
}

query {
	limit 100
	page  1
}

projects {

	variables {
		keycloak https://localhost:8080
		realm realm-test
	}

	auth basic {
		username foobar
		password tmp123!
	}

	get all {
		url /api/projects/	
	}

	post new-token {
		url `${keycloak}/realms/${realm}/openid-connect/token`

		before <<SCRIPT
			console.log("begin new token")
		SCRIPT

		after <<SCRIPT
			console.log("end new token")
		SCRIPT

		body urlencoded {
			grant_type password
			client_id my-client
			client_secret my-secret
			username $username
			password $password
		}

		compress true

		auth bearer "abc.erdfcv.xyz"
	}
}