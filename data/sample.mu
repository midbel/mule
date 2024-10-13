# this is a comment
# this is another comment

url http://localhost:8881

variables {
	sample @readfile data/sample.mu
}

flow geogeo {
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

 		after <<SCRIPT
 			console.log("[flow] done geo.countries")
 		SCRIPT

 		when 200 goto geo.continents
 		when 400
 		when 403 401 500
 	}
 	geo.continents {

 		after <<SCRIPT
 			console.log("[flow] done geo.continents")
 		SCRIPT

 		when 200
 		when 400
 		when 403 401 500
 	}
 }

get animals {
	url /animals/

	query {
		length 121
	}

	before <<SCRIPT
	SCRIPT

	after <<SCRIPT
	console.log(mule.response.body)
	SCRIPT

	get animalsWithBasic {
		auth basic {
			username foobar
			password tmp123!
		}
		headers {
			accept application/json
			referer localhost:9000
			accept-language fr nl en
		}
	}

	get animalsWithJwt {
		auth jwt {
			iss   mule.org
			user  foobar
			roles adm dev
		}
		headers {
			accept application/json
			referer localhost:9000
			Accept-Language fr nl en
		}
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

@include data/auth.mu