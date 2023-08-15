headers {
	accept          application/json
	content-type    application/json
}

variables {
	version 'v1.0.1'
	endpoint demo
}

collection check {}

collection abitofall {}

collection test {

	headers {
		accept-encoding gzip
		cache-control   no-cache
	}

	url http://localhost:9090/

	get demo {

		url ${endpoint}

		depends 'test.test'

		query {
			format ${version}
		}

		before <<BEFORE
		console.log(mule.collections)
		console.log(`start request ${requestName} to ${requestUri}`)
		BEFORE

		after <<AFTER
		console.log(`done request ${requestName} ${requestStatus}`)
		const obj = JSON.parse(responseBody)
		console.log(obj.id)
		AFTER

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