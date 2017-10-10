package main

import (
	"math"
	"sort"
	"strconv"

	jlexer "github.com/mailru/easyjson/jlexer"
	"github.com/valyala/fasthttp"
)

func EntityById(ctx *fasthttp.RequestCtx, entity string, idStr string) []byte {
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}
	switch entity {
	case "users":
		if id < int64(len(users)) && users[id] != nil {
			data, _ := users[id].MarshalJSON()
			return data
		}
	case "locations":
		if id < int64(len(locations)) && locations[id] != nil {
			data, _ := locations[id].MarshalJSON()
			return data
		}
	case "visits":
		if id < int64(len(visits)) && visits[id] != nil {
			data, _ := visits[id].MarshalJSON()
			return data
		}
	}
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	return nil
}

type visitPredicate func(*Visit) bool

func Visits(ctx *fasthttp.RequestCtx, idStr string) []byte {
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || id > int64(len(users)) || users[id] == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}
	user, filters := users[id], make([]visitPredicate, 0)
	args := ctx.QueryArgs()
	if args.Has("fromDate") {
		fromDate, err := args.GetUint("fromDate")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.VisitedAt > fromDate
		})
	}
	if args.Has("toDate") {
		toDate, err := args.GetUint("toDate")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.VisitedAt < toDate
		})
	}
	country := string(args.Peek("country"))
	if country != "" && len(country) > 0 {
		filters = append(filters, func(x *Visit) bool {
			return x.locationRef.Country == country
		})
	}
	if args.Has("toDistance") {
		toDistance, err := args.GetUint("toDistance")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.locationRef.Distance < toDistance
		})
	}
	resultVisits := make([]VisitResult, 0)
	for _, visit := range user.visits {
		satisfy := true
		for _, fn := range filters {
			if !fn(visit) {
				satisfy = false
				break
			}
		}
		if satisfy {
			resultVisits = append(resultVisits, VisitResult{
				Place:     visit.locationRef.Place,
				Mark:      visit.Mark,
				VisitedAt: visit.VisitedAt,
			})
		}
	}
	sort.Slice(resultVisits, func(i, j int) bool {
		return resultVisits[i].VisitedAt < resultVisits[j].VisitedAt
	})
	bytes, _ := VisitsResult{Visits: resultVisits}.MarshalJSON()
	return bytes
}

func Avg(ctx *fasthttp.RequestCtx, idStr string) []byte {
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || id > int64(len(locations)) || locations[id] == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}
	location, filters := locations[id], make([]visitPredicate, 0)
	args := ctx.QueryArgs()
	if args.Has("fromDate") {
		fromDate, err := args.GetUint("fromDate")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.VisitedAt > fromDate
		})
	}
	if args.Has("toDate") {
		toDate, err := args.GetUint("toDate")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.VisitedAt < toDate
		})
	}
	if args.Has("gender") {
		gender := string(args.Peek("gender"))
		if gender != "f" && gender != "m" {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.userRef.Gender == gender
		})
	}
	if args.Has("fromAge") {
		fromAge, err := args.GetUint("fromAge")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.userRef.Age >= fromAge
		})
	}
	if args.Has("toAge") {
		toAge, err := args.GetUint("toAge")
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return x.userRef.Age < toAge
		})
	}

	var count = 0
	var sum = 0
	for _, visit := range location.visits {
		satisfy := true
		for _, fn := range filters {
			if !fn(visit) {
				satisfy = false
				break
			}
		}
		if satisfy {
			count++
			sum += visit.Mark
		}
	}
	avg := 0.0
	if count > 0 {
		avg = float64(sum) / float64(count)
	}

	res := math.Pow(10, float64(5))
	avg = float64(round(avg*res)) / res
	bytes, _ := AvgResult{Avg: avg}.MarshalJSON()
	return bytes
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func Create(ctx *fasthttp.RequestCtx, entity string) []byte {
	switch entity {
	case "users":
		var user User
		err := user.UnmarshalJSON(ctx.PostBody())
		if err != nil || !user.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		users[user.ID] = &user
	case "locations":
		var location Location
		err := location.UnmarshalJSON(ctx.PostBody())
		if err != nil || !location.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		locations[location.ID] = &location
	case "visits":
		var visit Visit
		err := visit.UnmarshalJSON(ctx.PostBody())
		if err != nil || !visit.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		visits[visit.ID] = &visit
		location := locations[visit.Location]
		location.visits[visit.ID] = &visit
		visit.locationRef = location
		user := users[visit.User]
		user.visits[visit.ID] = &visit
		visit.userRef = user
	}

	return emptyJSON
}

func Update(ctx *fasthttp.RequestCtx, entity string, idStr string) []byte {
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return nil
	}
	switch entity {
	case "users":
		if id < int64(len(users)) && users[id] != nil {
			user := users[id]
			birthDate, email, firstName, lastName, gender := false, false, false, false, false
			var update User
			in := jlexer.Lexer{Data: ctx.PostBody()}
			in.Delim('{')
			for !in.IsDelim('}') {
				key := in.UnsafeString()
				in.WantColon()
				if in.IsNull() {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				switch key {
				case "id":
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case "birth_date":
					update.BirthDate = int(in.Int())
					birthDate = true
				case "email":
					update.Email = string(in.String())
					email = true
				case "first_name":
					update.FirstName = string(in.String())
					firstName = true
				case "last_name":
					update.LastName = string(in.String())
					lastName = true
				case "gender":
					update.Gender = string(in.String())
					gender = true
				default:
					in.SkipRecursive()
				}
				in.WantComma()
			}
			if !in.Ok() {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return nil
			}
			if birthDate {
				user.BirthDate = update.BirthDate
				user.CalculateAge()
			}
			if email {
				user.Email = update.Email
			}
			if firstName {
				user.FirstName = update.FirstName
			}
			if lastName {
				user.LastName = update.LastName
			}
			if gender {
				user.Gender = update.Gender
			}
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	case "locations":
		if id < int64(len(locations)) && locations[id] != nil {
			var update Location
			location := locations[id]
			distance, place, country, city := false, false, false, false
			in := jlexer.Lexer{Data: ctx.PostBody()}
			in.Delim('{')
			for !in.IsDelim('}') {
				key := in.UnsafeString()
				in.WantColon()
				if in.IsNull() {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				switch key {
				case "id":
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case "distance":
					update.Distance = int(in.Int())
					distance = true
				case "place":
					update.Place = string(in.String())
					place = true
				case "country":
					update.Country = string(in.String())
					country = true
				case "city":
					update.City = string(in.String())
					city = true
				default:
					in.SkipRecursive()
				}
				in.WantComma()
			}
			if !in.Ok() {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return nil
			}
			if distance {
				location.Distance = update.Distance
			}
			if place {
				location.Place = update.Place
			}
			if country {
				location.Country = update.Country
			}
			if city {
				location.City = update.City
			}
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	case "visits":
		if id < int64(len(visits)) && visits[id] != nil {
			var update Visit
			visit := visits[id]
			location, user, visitedAt, mark := false, false, false, false
			in := jlexer.Lexer{Data: ctx.PostBody()}
			in.Delim('{')
			for !in.IsDelim('}') {
				key := in.UnsafeString()
				in.WantColon()
				if in.IsNull() {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				switch key {
				case "id":
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case "location":
					update.Location = int(in.Int())
					location = visit.Location != update.Location
				case "user":
					update.User = int(in.Int())
					user = visit.User != update.User
				case "visited_at":
					update.VisitedAt = int(in.Int())
					visitedAt = true
				case "mark":
					update.Mark = int(in.Int())
					mark = true
				default:
					in.SkipRecursive()
				}
				in.WantComma()
			}
			if !in.Ok() {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return nil
			}
			if location {
				if len(locations) < update.Location || locations[update.Location] == nil {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				delete(visit.locationRef.visits, visit.ID)
				visit.Location = update.Location
				visit.locationRef = locations[update.Location]
				visit.locationRef.visits[visit.ID] = visit
			}
			if user {
				if len(users) < update.User || users[update.User] == nil {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				delete(visit.userRef.visits, visit.ID)
				visit.User = update.User
				visit.userRef = users[update.User]
				visit.userRef.visits[visit.ID] = visit
			}
			if visitedAt {
				visit.VisitedAt = update.VisitedAt
			}
			if mark {
				visit.Mark = update.Mark
			}
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	return nil
}

var emptyJSON = []byte("{}")
