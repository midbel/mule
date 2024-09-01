console.log("foobar", 100, true, [1, 2])
console.log({name: "foobar", age: 100})

let obj = JSON.parse('{"name": "foobar", "age": 100}')
JSON.stringify(obj)

function fib(n, x = 10) {
	console.log("n = ", n)
	console.log("x = ", x)
	return n + 1
}

fib(100, 10)

obj = {
	name: "foobar",
	age: 100,
	test: function() {
		console.log(this)
		return "testmeifyoucan"
	}
}

console.log(obj.test())