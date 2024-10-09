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

	variables {

	}

	query {

	}

	url http://localhost:8080

	before     ""
	beforeEach ""
	after      ""
	afterEach  ""
	
 	req1 {
 		before ""
 		after  ""
 		when 200 {
			set    local_var1 req1.body.var1
			set    local_var2 req1.body.var2
			unset  local_var2
			set    req2.url req1.url
			goto   req2
 		}
 		when 400 {
 			goto req3
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

	get countries {
		url /countries/
	}

	get continents {
		url /continents/
	}

}

@include data/auth.mu