#!/bin/sh

mkdir /data
unzip /tmp/data/data.zip -d /data
cp /tmp/data/options.txt /data

warmup () {
    sleep 30

    for i in {1..1000}; do
        curl -s -o /dev/null http://127.0.0.1/users/$i
        curl -s -o /dev/null http://127.0.0.1/locations/$i
        curl -s -o /dev/null http://127.0.0.1/visists/$i
        curl -s -o /dev/null http://127.0.0.1/users/$i/visits?toDistance=13
        curl -s -o /dev/null http://127.0.0.1/locations/$i/avg?gender=m
        curl -s -o /dev/null -H "Content-Type: application/json" -X POST -d '{"id":"0"}' http://127.0.0.1/users/new
        curl -s -o /dev/null -H "Content-Type: application/json" -X POST -d '{"id":"0"}' http://127.0.0.1/users/$i
    done

    curl -s -o /dev/null http://127.0.0.1/users/10000000
    curl -s -o /dev/null http://127.0.0.1/users/100000000000000
}

warmup & ./app