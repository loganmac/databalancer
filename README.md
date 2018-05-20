# Data Balancer

This repository is a programming challenge. You're building a micro-service which will ingest data of arbitrary shape and type and balance the incoming data across strongly typed MySQL tables so that the data can be queried further. Read the following [Current Code](#current-code) section to understand what the `main.go` file of this repository is doing. Alternatively, or additionally, read `main.go` to understand what the `main.go` file of this repository is doing. Then, read the [Objectives](#objectives) section of this README to understand the challenge expectations.

As you develop code, create branches and submit pull requests to this repository. Names for pull requests will be mentioned in the instructions below. Merge these PRs as you go and when you are done, we'll use them to review the code.

## Current Code

When you run the `databalancer` binary, you will see output like:

```
$ databalancer -mysql_address="192.168.99.100:3306"

2017/01/06 20:47:32 Connected to MySQL as root at 192.168.99.100:3306
2017/01/06 20:47:32 Starting HTTP server on :8080
```

### Ingest Endpoint

One of the endpoints that exists at the moment is the IngestLog endpoint at `/api/log`. The IngestLog endpoint expects a `HTTP PUT` request with a JSON body in the following format:

```go
// IngestLogBody is the format of the JSON required in the body of a request to
// the IngestLogHandler
type IngestLogBody struct {
	Family string                   `json:"family"`
	Schema map[string]string        `json:"schema"`
	Logs   []map[string]interface{} `json:"logs"`
}
```

An example request body to `PUT /api/log` might be:

```json
{
  "family": "dog_registry",
  "schema": {
    "name": "string",
    "breed": "string",
    "weight": "int"
  },
  "logs": [
    {
      "name": "spot",
      "breed": "labrador",
      "weight": 100
    },
    {
      "name": "max",
      "breed": "chihuahua",
      "weight": 3
    },
    {
      "name": "sprinkle",
      "breed": "pitbull",
      "weight": 50
    }
  ]
}
```

As you can see, the body defines:

- The "log family", which in this case is "dog_registry"
- The schema of the fields that will be logged in each "log event"
- A list of log events

A request may include 0 or more "log events", all of which are apart of the same "log family". Every request will include the log family name and the schema of each event's fields.

If you're running the `databalancer` server on `localhost:8080`, you should be able to send the following request:

```
curl \
  -H "Content-Type: application/json" \
  -X PUT \
  -d '{"family":"dog_registry","schema":{"name":"string","breed":"string","weight":"int"},"logs":[{"name":"spot","breed":"labrador","weight":100},{"name":"max","breed":"chihuahua","weight":3},{"name":"sprinkle","breed":"pitbull","weight":50}]}' \
  http://localhost:8080/api/log
```

If this request is received by the server properly, the running server process will emit the following logs:

```
2017/01/06 20:48:32 Received logs for the dog_registry log family
2017/01/06 20:48:32 Log values for the field name of the dog_registry log will be of type string
2017/01/06 20:48:32 Log values for the field breed of the dog_registry log will be of type string
2017/01/06 20:48:32 Log values for the field weight of the dog_registry log will be of type int
2017/01/06 20:48:32 Handling a new log event for the dog_registry log family
2017/01/06 20:48:32 The value of the breed field in the dog_registry log event is labrador
2017/01/06 20:48:32 The value of the weight field in the dog_registry log event is 100
2017/01/06 20:48:32 The value of the name field in the dog_registry log event is spot
2017/01/06 20:48:32 Handling a new log event for the dog_registry log family
2017/01/06 20:48:32 The value of the name field in the dog_registry log event is max
2017/01/06 20:48:32 The value of the breed field in the dog_registry log event is chihuahua
2017/01/06 20:48:32 The value of the weight field in the dog_registry log event is 3
2017/01/06 20:48:32 Handling a new log event for the dog_registry log family
2017/01/06 20:48:32 The value of the name field in the dog_registry log event is sprinkle
2017/01/06 20:48:32 The value of the breed field in the dog_registry log event is pitbull
2017/01/06 20:48:32 The value of the weight field in the dog_registry log event is 50
```

Additionally, if you query the `raw_logs` table in the MySQL database after this request, you should see the following results:

```
mysql> select * from raw_logs;
+----+--------------+---------------------------------------------------+
| id | family       | log                                               |
+----+--------------+---------------------------------------------------+
|  1 | dog_registry | {"breed":"labrador","name":"spot","weight":100}   |
|  2 | dog_registry | {"breed":"chihuahua","name":"max","weight":3}     |
|  3 | dog_registry | {"breed":"pitbull","name":"sprinkle","weight":50} |
+----+--------------+---------------------------------------------------+
3 rows in set (0.00 sec)
```

If you execute that same HTTP request again and issue the same query to MySQL, you should see the following results:

```
mysql> select * from raw_logs;
+----+--------------+---------------------------------------------------+
| id | family       | log                                               |
+----+--------------+---------------------------------------------------+
|  1 | dog_registry | {"breed":"labrador","name":"spot","weight":100}   |
|  2 | dog_registry | {"breed":"chihuahua","name":"max","weight":3}     |
|  3 | dog_registry | {"breed":"pitbull","name":"sprinkle","weight":50} |
|  4 | dog_registry | {"breed":"labrador","name":"spot","weight":100}   |
|  5 | dog_registry | {"breed":"chihuahua","name":"max","weight":3}     |
|  6 | dog_registry | {"breed":"pitbull","name":"sprinkle","weight":50} |
+----+--------------+---------------------------------------------------+
6 rows in set (0.00 sec)
```

### Query Endpoint

Another API endpoint is the Query endpoint at `/api/query`. The Query endpoint expects a `HTTP POST` request with a JSON body in the following format:

```go
// QueryBody is the format of the JSON required in the body of a request to the QueryHandler
type QueryBody struct {
	Query string `json:"query"`
}
```

An example request body to `POST /api/query` might be:

```json
{
  "query": "SELECT * FROM `dog_registry`;"
}
```

As you can see, the body defines:

- The "query", which is a SQL SELECT query string

If you're running the `databalancer` server on `localhost:8080`, you should be able to send the following request:

```
curl \
  -H "Content-Type: application/json" \
  -X POST \
  -d '{"query":"SELECT * FROM `dog_registry`;"}' \
  http://localhost:8080/api/query
```

If this request is received by the server properly, the running server process should return something like:

```json
{
  "results": [
    {
      "breed":"labrador",
      "id":1,
      "name":"spot",
      "weight":100
    },
    {
      "breed":"chihuahua",
      "id":2,
      "name":"max",
      "weight":3
    },
    {
      "breed":"pitbull",
      "id":3,
      "name":"sprinkle",
      "weight":50
    }
  ]
}
```

### Describe Endpoint

Yet another API endpoint is the Describe endpoint at `/api/describe`. The Describe endpoint expects a `HTTP GET` request.

If you're running the `databalancer` server on `localhost:8080`, you should be able to send the following request:

```
curl \
  -H "Content-Type: application/json" \
  -X GET \
  http://localhost:8080/api/describe
```

If this request is received by the server properly, the running server process should return something like:

```json
{
  "tables": [
    {
      "columns": [
        {
          "name": "id",
          "nullable": false,
          "type": "int"
        },
        {
          "name": "age",
          "nullable": true,
          "type": "int"
        },
        {
          "name": "breed",
          "nullable": true,
          "type": "text"
        },
        {
          "name": "name",
          "nullable": true,
          "type": "text"
        }
      ],
      "name": "cat_registry"
    },
    {
      "columns": [
        {
          "name": "id",
          "nullable": false,
          "type": "int"
        },
        {
          "name": "name",
          "nullable": true,
          "type": "text"
        },
        {
          "name": "breed",
          "nullable": true,
          "type": "text"
        },
        {
          "name": "weight",
          "nullable": true,
          "type": "int"
        }
      ],
    "name": "dog_registry"
    }
  ]
}
```

## Objectives

### Dynamic table creation and logging

Right now, the `raw_logs` table is created statically when the `databalancer` binary starts and all log events are stored in that table. The objective here is to enable more complex analysis on this data after (and during) it's ingestion. To enable this, you must dynamically create a new table for each log family (if there isn't already one created) as logs are streamed into your service.

For the above example, there should be a `dog_registry` table automatically created with "breed", "name", and "weight" as top-level columns. Thus, I should be able to execute queries like:

```
mysql> select * from dog_registry where breed = "labrador" and name != "max" order by weight;
```

Keep in mind that `databalancer` may be run on many web servers, so all web requests must be stateless. Authentication is outside the scope of this exercise.

Do this and create a pull request called `dynamic-tables`. Make note of any compromises/ considerations in your code due to the time constraint.

Merge this PR. 

### Query API

Once dynamic tables are created, we need to expose the ability to users to query the datasets. Add a new HTTP endpoint which accepts a SQL query and returns the results. Document the request and response format.

Once you've added APIs for querying the datasets, create a pull request called `query-api`. Make note of any compromises/ considerations in your code due to the time constraint.

Merge this PR.

### Multiple MySQL databases

Use Docker and Docker Compose to add a few more MySQL databases. For now, assume that all of the log_events in a single log family can fit on a single database, but the total data being streamed from all log families will not. Thus, you must load balance between 2 or more MySQL database servers, while keeping track of what data is where. Update the query API to support this new functionality.

Once your application supports streaming logs into a sharded MySQL, create a pull request called `mysql-sharding`. Make note of any compromises/ considerations in your code due to the time constraint.

Merge this PR. 

### Bonus

If that all is too easy, consider the following questions and implement solutions whenever possible:

- What information would be needed to automatically purge out old results and what endpoints would be needed to facilitate that process?
- What if the results of the data in a single log family are too big to fit on one database server?
- How could one make a service like `databalancer` more secure? What would the trade-offs be?
- How could you be better testing your code?
- Run some rigorous benchmarks on your service. Reason about the implications of your benchmarks for running your code in a production environment.

## Installing dependencies

There's a `deps` target in the makefile, so just run `make deps`.

For your convenience, a `docker-compose.yml` file is included with this repository. [Install Docker](https://www.docker.com/products/overview#/install_the_platform) and run `docker-compose up` from the root of the repository to use [Docker Compose](https://docs.docker.com/compose/) to manage the necessary infrastructure.

To run the service against a distributed database (TiDB), run `make db_start`, then wait a few seconds and run `make db_setup`. To stop, run `make db_stop`.

## Databases and other infrastructure

Use [Docker](https://docs.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) to manage your infrastructure.

Currently, running `docker-compose up` from the root of this repository will pull down the MySQL image from Docker Hub, create a default user account, and run the `mysqld` service, mapping port 3306 on the MySQL server to port 3306 on your localhost. If you use [Docker Toolbox](https://www.docker.com/products/docker-toolbox) instead of a native Docker engine, the IP address that Docker maps ports to will not be `localhost`, it will be whatever the output of `docker-machine ip` is. In this case, make sure that you specify that hostname when you run the `databalancer` binary. For example:

```
databalancer -mysql_address="192.168.99.100:3306"
```

If you'd like to change any of the other MySQL connection parameters, run `databalancer -help` for a complete list of configurable connection options, or modify the code yourself to support your desires.

## Building and running the code

Make sure you have done the following:

- Cloned this repository into the correct location in your `$GOPATH`
- Installed all required dependencies (ie: Docker)
- Ran `docker-compose up` to start the virtual infrastructure

If so, then you are ready to build the code via:

```
make build
```

This should create a binary called `databalancer` if you're using OS X or Linux and `databalancer.exe` if you're using Windows. To run the program, simple run the created binary.

Execute `databalancer -help` for options:

```
$ databalancer -help
Usage of C:\Users\marpaia\go\src\github.com\kolide\databalancer\databalancer.exe:
  -mysql_address string
        The MySQL server address (default "localhost:3306")
  -mysql_database string
        The MySQL database to use (default "databalancer")
  -mysql_password string
        The MySQL user account password (default "")
  -mysql_username string
        The MySQL user account username (default "root")
  -server_address string
        The address and port to serve the local HTTP server (default ":8080")
```
