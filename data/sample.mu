# this is a comment
# this is another comment

url http://localhost:8001

variables {
	sample @readfile data/sample.mu
}

flow name1 {
	headers {
		content-type application/json
	}
	
 	req1 {
 		when 200 goto req2 {
			set    local_var1 req1.body.var1
			set    local_var2 req1.body.var2
			unset  local_var2
			set    req2.url req1.url
 		}
 		when 400 goto req3 {
 			exit
 		}
 		when 403 401 500 {
 			exit
 		}
 	}
 	req2 {
 
 	}
 	req3 {
 
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
			console.log("after", mule.response.body)
		SCRIPT
	}

	get continents {
		url /continents/
	}

}

@include data/auth.mu