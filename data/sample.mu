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
		(mule.response.json() || []).forEach(c => console.log(c.id, c.code, c.name))
		console.log(mule.response.headers.entries())
		SCRIPT
	}

	get continents {
		url /continents/
	}

}

@include data/auth.mu