variables {
	accessToken "supersecrettoken11!"
	test01 $accessToken
	test02 ${accessToken}
	test03 ${accessToken:1:4}
	test04 ${accessToken::4}
	test05 ${accessToken:1:}
	test06 ${accessToken#super}
	test07 ${accessToken##super}
	test08 ${accessToken%"11!"}
	test09 ${accessToken%%"11!"}
	test10 ${accessToken/super/mega}
	test11 ${accessToken//e/E}
	test12 ${accessToken/#super/top}
	test13 ${accessToken/%"11!"/"21!"}
	test14 ${accessToken,}
	test14 ${accessToken,,}
	test14 ${accessToken^}
	test14 ${accessToken^^}
}