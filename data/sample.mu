# this is a comment
# this is another comment

url http://localhost:8001

variables {
	sample @readfile data/sample.mu
}

# flow name1 {
# 	req1 {
# 		when 200 {
#			goto req2
# 		}
# 		when 400 {
# 			goto req3
# 		}
# 		when 403, 401, 500 {
# 			exit
# 		}
# 	}
# 	req2 {
# 
# 	}
# 	req3 {
# 
# 	}
# }

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