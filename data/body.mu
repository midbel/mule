url "http://localhost:8881"

variables {
	answer 42
}

post xml {
	url "/dump"

	body xml {
		repositories {
			repo {
				_id mule
				_status active
				_star $answer
				name mule
				author midbel
				language js
				language go
			}
			repo {
				_id tish
				_status waiting
				_star $answer
				name tish
				author midbel
				language go
			}
			repo {
				_id sweet
				_status waiting
				_star $answer
				name sweet
				author midbel
				language go
			}
		}
	}
}