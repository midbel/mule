collection test {

	headers {
		accept          application/json
		content-type    application/json
		accept-encoding gzip
		cache-control   no-cache
	}

	get demo {

		depends 'test.test'

		url http://localhost:9090/demo

		query {
			format v1
		}

		username test
		password test

	}

	get test {

		url http://localhost:9090/test?format=1

		query {
			dtstart '2023-01-01'
			dtend   '2024-01-01'
		}

		username test
		password test

		headers {
			Authorization "bearer 123"
			User-Agent     'mule 0.1' 
		}

	}

}

variables {

	version 1
	token   '0123456789FEDCBA'
}

get demo {

	url http://localhost:8000
	retry 1

	headers {
		authentication "bearer ${token}"
		content-type   text/csv
	}

	query {
		format ${version}
	}

	body <<EOF
	EOF

}