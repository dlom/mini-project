Mini Project
============

## To run
```
docker-compose up -d [--build]
```

## To run the testing script
```
node tester
```

## Scaling up

If a single instance of the ingestion service cannot properly handle the load, it can be easily scaled using `docker-compose`:
```
docker-compose up -d --scale ingestion=10
```
This usually manifests itself in `500: Internal Server Error` responses.

## Rationales

* The entire stringified JSON is pushed into the redis queue because it does not seem to affect performance.  Using the `redis-benchmark` tool, I saw that performance for `POP`ing and `PUSH`ing remained the same from 3 byte payloads all the way up to 3kB payloads.
