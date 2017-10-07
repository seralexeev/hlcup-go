package main

import "github.com/valyala/fasthttp"

func EntityById(ctx *fasthttp.RequestCtx) []byte {
	data, _ := users[1].MarshalJSON()
	ctx.SetContentType("application/json")
	return data
}
