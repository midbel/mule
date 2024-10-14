url http://localhost:8881

variables {
	accessToken "supersecrettoken11!"
}

post "token" {
	url /token/new

	body json {
		user "foobar"
		pass "tmp123!"
		grant read
		grant write
	}

	expect 200

	after <<SCRIPT
	const res = mule.response.json()
	mule.collection.set("accessToken", res.token)
	SCRIPT
}

get verify {
	url /token

	body json {
		token ${accessToken}
	}

	expect 204
}

flow check {
	"token" {
		when 200 goto verify
	}
	verify {
		when 204
	}
}