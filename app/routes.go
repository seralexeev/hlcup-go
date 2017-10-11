package main

import (
	"bytes"
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
	switch entity[0] {
	case 'u':
		if id < int64(len(users)) && users[id] != nil {
			data, _ := users[id].MarshalJSON()
			return data
		}
	case 'l':
		if id < int64(len(locations)) && locations[id] != nil {
			data, _ := locations[id].MarshalJSON()
			return data
		}
	case 'v':
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
	country := args.Peek("country")
	if len(country) > 0 {
		filters = append(filters, func(x *Visit) bool {
			return bytes.Equal(x.locationRef.Country, country)
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
		if visit == nil {
			continue
		}
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
		gender := args.Peek("gender")
		if !bytes.Equal(gender, f) && !bytes.Equal(gender, m) {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		filters = append(filters, func(x *Visit) bool {
			return bytes.Equal(x.userRef.Gender, gender)
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
		if visit == nil {
			continue
		}
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
	switch entity[0] {
	case 'u':
		user := new(User)
		err := user.UnmarshalJSON(ctx.PostBody())
		if err != nil || !user.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		user.visits = make([]*Visit, 10)
		users[user.ID] = user
	case 'l':
		location := new(Location)
		err := location.UnmarshalJSON(ctx.PostBody())
		if err != nil || !location.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		location.visits = make([]*Visit, 10)
		locations[location.ID] = location
	case 'v':
		visit := new(Visit)
		err := visit.UnmarshalJSON(ctx.PostBody())
		if err != nil || !visit.IsValid() {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return nil
		}
		visits[visit.ID] = visit
		location := locations[visit.Location]
		location.visits = append(location.visits, visit)
		visit.locationRef = location
		user := users[visit.User]
		user.visits = append(user.visits, visit)
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
	switch entity[0] {
	case 'u':
		if id < int64(len(users)) && users[id] != nil {
			user := users[id]
			birthDate, email, firstName, lastName, gender := false, false, false, false, false
			update := new(User)
			in := jlexer.Lexer{Data: ctx.PostBody()}
			in.Delim('{')
			for !in.IsDelim('}') {
				key := in.UnsafeString()
				in.WantColon()
				if in.IsNull() {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				switch key[0] {
				case 'i':
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case 'b':
					update.BirthDate = int(in.Int())
					birthDate = true
				case 'e':
					update.Email = in.Bytes()
					email = true
				case 'f':
					update.FirstName = in.Bytes()
					firstName = true
				case 'l':
					update.LastName = in.Bytes()
					lastName = true
				case 'g':
					update.Gender = in.Bytes()
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
	case 'l':
		if id < int64(len(locations)) && locations[id] != nil {
			update := new(Location)
			location := locations[id]
			distance, place, country, city := false, false, false, false
			in := jlexer.Lexer{Data: ctx.PostBody()}
			in.Delim('{')
			for !in.IsDelim('}') {
				key := in.UnsafeString()
				key0 := key[0]
				in.WantColon()
				if in.IsNull() {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				switch {
				case key0 == 'i':
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case key0 == 'd':
					update.Distance = int(in.Int())
					distance = true
				case key0 == 'p':
					update.Place = in.Bytes()
					place = true
				case key0 == 'c' && key[1] == 'o':
					update.Country = in.Bytes()
					country = true
				case key0 == 'c' && key[1] == 'i':
					update.City = in.Bytes()
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
	case 'v':
		if id < int64(len(visits)) && visits[id] != nil {
			update := new(Visit)
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
				switch key[0] {
				case 'i':
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				case 'l':
					update.Location = int(in.Int())
					location = visit.Location != update.Location
				case 'u':
					update.User = int(in.Int())
					user = visit.User != update.User
				case 'v':
					update.VisitedAt = int(in.Int())
					visitedAt = true
				case 'm':
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
				for i, v := range visit.locationRef.visits {
					if v != nil && v.ID == visit.ID {
						// visit.locationRef.visits = append(visit.locationRef.visits[:i], visit.locationRef.visits[i+1:]...)
						visit.locationRef.visits[i] = nil
						break
					}
				}
				visit.Location = update.Location
				visit.locationRef = locations[update.Location]
				visit.locationRef.visits = append(visit.locationRef.visits, visit)
			}
			if user {
				if len(users) < update.User || users[update.User] == nil {
					ctx.SetStatusCode(fasthttp.StatusBadRequest)
					return nil
				}
				for i, v := range visit.userRef.visits {
					if v != nil && v.ID == visit.ID {
						// visit.userRef.visits = append(visit.userRef.visits[:i], visit.userRef.visits[i+1:]...)
						visit.userRef.visits[i] = nil
						break
					}
				}
				visit.User = update.User
				visit.userRef = users[update.User]
				visit.userRef.visits = append(visit.userRef.visits, visit)
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
var f = []byte("f")
var m = []byte("m")
