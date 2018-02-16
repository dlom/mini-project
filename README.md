Kochava Mini Project
====================

To run: `docker-compose up -d`

## Rationales

* The entire stringified JSON is pushed into the redis queue because it does not seem to affect performance.  Using the `redis-benchmark` tool, I saw that performance for `POP`ing and `PUSH`ing remained the same from 3 byte payloads all the way up to 3kB payloads.
