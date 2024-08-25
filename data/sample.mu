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

	get all {
		url /api/projects/	
		auth basic {
			username foobar
			password tmp123
		}
	}

	post new-token {
		url `${keycloak}/realms/${realm}/openid-connect/token`

		body urlencoded({
			grant_type password
			client_id my-client
			client_secret my-secret
			username $username
			password $password
		})

		auth bearer "abc.erdfcv.xyz"

		headers {
			content-type x-www-urlencoded
		}
	}
}