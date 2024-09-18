url http://localhost:8080

variables {
	realm poc
}

get access {
	url /realms/${realm}/openid-connect/token
}