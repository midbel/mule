# this is a comment
# this is another comment

url http://localhost:8001

variables {
	sample @readfile data/sample.mu
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
	}

	get animalsWithJwt {
		auth jwt {
			iss   mule.org
			user  foobar
			roles adm dev
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