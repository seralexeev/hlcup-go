package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/debug"
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
		if len(parts) < 3 || len(parts[1]) < 1 || len(parts[2]) < 1 {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}
		var body []byte
		p1 := parts[1][0]
		p2 := parts[2][0]
		l := len(parts)
		switch {
		case ctx.IsGet() && l == 4 && p1 == 'l' && len(parts[3]) > 0 && parts[3][0] == 'a':
			body = Avg(ctx, parts[2])
		case ctx.IsGet() && l == 3 && (p1 == 'u' || p1 == 'l' || p1 == 'v'):
			body = EntityById(ctx, p1, parts[2])
		case ctx.IsGet() && l == 4 && p1 == 'u' && len(parts[3]) > 0 && parts[3][0] == 'v':
			body = Visits(ctx, parts[2])
		case ctx.IsPost() && l == 3 && p2 == 'n' && (p1 == 'u' || p1 == 'l' || p1 == 'v'):
			body = Create(ctx, p1)
		case ctx.IsPost() && l == 3 && (p1 == 'u' || p1 == 'l' || p1 == 'v'):
			body = Update(ctx, p1, parts[2])
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}

		if body != nil && len(body) > 0 {
			ctx.Response.Header.SetContentLength(len(body))
			ctx.Response.Header.SetContentTypeBytes(contentTypeBytes)
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
	debug.SetGCPercent(50)
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
			usersFile := new(UsersFile)
			usersFile.UnmarshalJSON(data)
			for _, user := range usersFile.Users {
				users[user.ID] = user
				user.visits = make([]*Visit, 10)
				user.CalculateAge()
			}
		}
		if strings.HasPrefix(file.Name(), "locations") {
			data, _ := ioutil.ReadFile(path.Join(dir, file.Name()))
			locationsFile := new(LocationsFile)
			locationsFile.UnmarshalJSON(data)
			for _, location := range locationsFile.Locations {
				locations[location.ID] = location
				location.visits = make([]*Visit, 10)
			}
		}
		if strings.HasPrefix(file.Name(), "visits") {
			data, _ := ioutil.ReadFile(path.Join(dir, file.Name()))
			visitsFile := new(VisitsFile)
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

	runtime.GC()
	debug.SetGCPercent(-1)
}

var contentTypeBytes = []byte("application/json")
