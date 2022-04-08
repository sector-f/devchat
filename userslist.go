package main

import (
	"encoding/json"
	"io/ioutil"
)

type userList []userListEntry

func userListFromFile(filename string) (userList, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return userList{}, err
	}

	var u userList
	err = json.Unmarshal(file, &u)
	return u, err
}

type userListEntry struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}
