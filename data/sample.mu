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

	get all {
		url /api/projects/	
		username foobar
		password foobar
	}

	get token {
		url https://localhost:8080/realms/realm-test/openid/token
		body {
			grant_type password
			client_id my-client
			client_secret my-secret
			username $username
			password $password
		}
	}
}