package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
)

var users [1500200]*User
var locations [1000000]*Location
var visits [10500000]*Visit
var currentDate int

func main() {
	args := os.Args[1:]
	dataPath := "/data"
	if len(args) > 0 {
		dataPath = args[0]
	}
	port := "80"
	if len(args) > 1 {
		port = args[1]
	}

	fmt.Println(dataPath)
	fmt.Println(port)

	loadData(dataPath)

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		parts := strings.Split(path, "/")
		var body []byte
		switch {
		case ctx.IsGet() && len(parts) == 4 && parts[1] == "locations" && parts[3] == "avg":
			body = Avg(ctx, parts[2])
		case ctx.IsGet() && len(parts) == 3 && (parts[1] == "users" || parts[1] == "locations" || parts[1] == "visits"):
			body = EntityById(ctx, parts[1], parts[2])
		case ctx.IsGet() && len(parts) == 4 && parts[1] == "users" && parts[3] == "visits":
			body = Visits(ctx, parts[2])
		case ctx.IsPost() && len(parts) == 3 && parts[2] == "new" &&
			(parts[1] == "users" || parts[1] == "locations" || parts[1] == "visits"):
			body = Create(ctx, parts[1])
		case ctx.IsPost() && len(parts) == 3 &&
			(parts[1] == "users" || parts[1] == "locations" || parts[1] == "visits"):
			body = Update(ctx, parts[1], parts[2])
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}

		if body != nil && len(body) > 0 {
			ctx.Response.Header.SetContentLength(len(body))
			ctx.SetBody(body)
		}
	}

	err := fasthttp.ListenAndServe(":"+port, requestHandler)
	if err != nil {
		log.Fatal(err)
	}
}

func fileOrder(path string) int {
	if strings.HasPrefix(path, "users") {
		return 0
	}

	if strings.HasPrefix(path, "locations") {
		return 1
	}

	if strings.HasPrefix(path, "visits") {
		return 2
	}

	return 10
}

func loadData(dir string) {
	opts, err := ioutil.ReadFile(path.Join(dir, "options.txt"))
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(opts), "\n")
	currentDate, _ = strconv.Atoi(lines[0])
	fmt.Println(currentDate)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(files, func(i, j int) bool {
		return fileOrder(files[i].Name()) < fileOrder(files[j].Name())
	})

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "users") {
			data, _ := ioutil.ReadFile(path.Join(dir, file.Name()))
			var usersFile UsersFile
			usersFile.UnmarshalJSON(data)
			for _, user := range usersFile.Users {
				users[user.ID] = user
				user.CalculateAge()
			}
		}
		if strings.HasPrefix(file.Name(), "locations") {
			data, _ := ioutil.ReadFile(path.Join(dir, file.Name()))
			var locationsFile LocationsFile
			locationsFile.UnmarshalJSON(data)
			for _, location := range locationsFile.Locations {
				locations[location.ID] = location
			}
		}
		if strings.HasPrefix(file.Name(), "visits") {
			data, _ := ioutil.ReadFile(path.Join(dir, file.Name()))
			var visitsFile VisitsFile
			visitsFile.UnmarshalJSON(data)
			for _, visit := range visitsFile.Visits {
				visits[visit.ID] = visit
				location := locations[visit.Location]
				location.visits = append(location.visits, visit)
				visit.locationRef = location

				user := users[visit.User]
				user.visits = append(user.visits, visit)
				visit.userRef = user
			}
		}
	}
}
