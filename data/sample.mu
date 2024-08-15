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
}