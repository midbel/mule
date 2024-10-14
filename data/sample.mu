# this is a comment
# this is another comment

url http://localhost:8881

variables {
	accessToken "supersecrettoken11!"
}

flow earth {
	headers {
		content-type application/json
	}

	before <<BEFORE
		console.log(">>> geo: before")
	BEFORE

	after <<AFTER
		console.log(">>> geo: after")
	AFTER
	
 	geo.countries {

 		when 200 goto geo.continents
 		when 400
 		when 403 401 500
 	}
 	geo.continents {

 		when 200
 		when 400
 		when 403 401 500
 	}
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

flow checktoken {
	"token" {
		when 200 goto verify
	}
	verify {
		when 204
	}
}

errors {
	url /codes
	
	get badrequest {
		url    /400
		expect 400
	}

	get servererror {
		url /500
		expect 500
	}
}

get animals {
	url /animals/

	query {
		length 121
	}
}


geo {

	url http://localhost:8881

	headers {
		Content-Type application/json
		Accept       application/json
	}

	get countries {
		url /countries/

		before <<SCRIPT
			console.log("before", mule.request.url)
		SCRIPT

		after <<SCRIPT
			console.log("after", mule.request.url)
			console.log("after", mule.response.code)
			console.log("after", mule.response.success())
			console.log("after", mule.response.fail())
			console.log("after", mule.response.json())
		SCRIPT
	}

	get continents {
		url /continents/

		before <<SCRIPT
			console.log("before", mule.request.url)
		SCRIPT

		after <<SCRIPT
			console.log("after", mule.request.url)
			console.log("after", mule.response.code)
			console.log("after", mule.response.success())
			console.log("after", mule.response.fail())
			console.log("after", mule.response.json())
		SCRIPT
	}

}