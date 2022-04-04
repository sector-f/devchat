package main

type config struct {
	port        int
	scrollback  int
	banFilename string
}

func defaultConfig() config {
	c := config{
		port:        22,
		scrollback:  100,
		banFilename: "bans.json",
	}

	return c
}
