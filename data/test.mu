headers {
  content-type "application/json"
  accept "text/xml"
}

username foobar
password "tmp123!"

collection demo {

  url "http://localhost:9001"

  get animals {

    url "/animals/"
    headers {
      accept "text/xml;q=0.7"
      accept "application/json;q=0.8"
    }

    query {
      offset 5
      count  5
    }

    expect 200

    before <<SCRIPT
    console.log("query:", requestName)
    SCRIPT

    after <<SCRIPT
    console.log("done query:", requestName, responseStatus)
    console.log("response body", responseBody)
    SCRIPT
  }

  get colors {
    url "/colors/"
  }

  get cars {
    url "/cars/"

    depends "demo.colors"

    query {
      count 3
    }
  }
}
