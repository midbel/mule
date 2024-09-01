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
	},
	test2() {
		console.log("test2")
	}
}

console.log(obj.test())
console.log(obj.test2())

fib2(function() {
	return 1
})