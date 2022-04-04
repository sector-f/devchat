package main

type config struct {
	port        int
	scrollback  int
	banFilename string
	keyFilename string
}

func defaultConfig() config {
	c := config{
		port:        2222,
		scrollback:  100,
		banFilename: "bans.json",
		keyFilename: "id_rsa",
	}

	return c
}
