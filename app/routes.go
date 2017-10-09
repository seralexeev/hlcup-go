package main

import (
	"math"
	"sort"
	"strconv"

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
		location.visits = append(location.visits, &visit)
		visit.locationRef = location
		user := users[visit.User]
		user.visits = append(user.visits, &visit)
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
			users[id].UnmarshalJSON(ctx.PostBody())
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	case "locations":
		if id < int64(len(locations)) && locations[id] != nil {
			locations[id].UnmarshalJSON(ctx.PostBody())
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	case "visits":
		if id < int64(len(visits)) && visits[id] != nil {
			visits[id].UnmarshalJSON(ctx.PostBody())
			return emptyJSON
		}
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	return nil
}

var emptyJSON = []byte("{}")
