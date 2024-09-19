# this is a comment
# this is another comment

url http://localhost:8001

variables {
	sample @readfile data/sample.mu
}

get animals {
	url /animals/
}

geo {
	get countries {
		url /countries/

		after <<SCRIPT
			const all = mule.response
			console.log(all)
		SCRIPT
	}

	get continents {
		url /continents/
	}

}

@include data/auth.mu