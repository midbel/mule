# this is a comment
# this is another comment

url http://localhost:3000
username foobar

variables {
	token abcdef
}

query {
	limit 100
	page  1
}

projects {
	password demo-test

	variables {
		keycloak https://localhost:8080
		realm realm-test
	}

	get all {
		url /api/projects/	
		username foobar
		password foobar
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

		headers {
			content-type x-www-urlencoded
		}
	}
}