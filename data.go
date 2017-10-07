package main

import "github.com/buger/jsonparser"

type User struct {
	Id, age, birth_date                  int
	email, first_name, last_name, gender []byte

	visits []*Visit
}

func readUser(data []byte) (*User, error) {
	var user User
	jsonparser.EachKey(data, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		switch idx {
		case 0:
			v, _ := jsonparser.ParseInt(value)
			user.Id = int(v)
		case 1:
			v, _ := jsonparser.ParseInt(value)
			user.birth_date = int(v)
		case 2:
			user.email = value
		case 3:
			user.first_name = value
		case 4:
			user.last_name = value
		case 5:
			user.gender = value
		}
	}, [][]string{
		[]string{"id"},
		[]string{"birth_date"},
		[]string{"email"},
		[]string{"first_name"},
		[]string{"last_name"},
		[]string{"gender"},
	}...)

	return &user, nil
}

type Location struct {
	id, distance         int
	place, country, city []byte

	visits []*Visit
}

type Visit struct {
	id, location, user, visited_at, mark int

	userRef     *User
	locationRef *Location
}
