url http://localhost:8881/dump

variables {
	answer 42
}

auth jwt {
	name foobar
	age  42
	roles dev
	roles adm
	iss  http://foobar.org
}

post json {
	body json {
		languages go
		languages js
		name mule
		developer {
			name midbel
			mail "midbel@foobar.org"
			org  foobar
		}
		repo http://gitea.foobar.org/midbel/mule
	}
}

post xml {
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
		}
	}
}