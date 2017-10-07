package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/valyala/fasthttp"
)

var users [1500200]*User

func main() {
	loadData()

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		parts := strings.Split(path, "/")

		fmt.Println(parts)
		var body []byte
		switch {
		case ctx.IsGet() && len(parts) == 3 && parts[1] == "users" || parts[1] == "locations" || parts[1] == "visits":
			body = EntityById(ctx)
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}

		if len(body) > 0 {
			ctx.Response.Header.SetContentLength(len(body))
			ctx.SetBody(body)
		}
	}

	fasthttp.ListenAndServe(":8081", requestHandler)
}

func loadData() {
	data, _ := ioutil.ReadFile("/Users/sergey/projects/hlcupdocs/data/TRAIN/data/users_1.json")
	jsonparser.ArrayEach(data, func(item []byte, dataType jsonparser.ValueType, offset int, err error) {
		user, _ := readUser(item)
		users[user.Id] = user
		fmt.Println(users[user.Id])

	}, "users")
}
