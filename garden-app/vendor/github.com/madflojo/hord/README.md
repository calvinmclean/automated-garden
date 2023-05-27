# Hord

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/madflojo/hord)
[![codecov](https://codecov.io/gh/madflojo/hord/branch/main/graph/badge.svg?token=0TTTEWHLVN)](https://codecov.io/gh/madflojo/hord)
[![Go Report Card](https://goreportcard.com/badge/github.com/madflojo/hord)](https://goreportcard.com/report/github.com/madflojo/hord)
[![Documentation](https://godoc.org/github.com/madflojo/hord?status.svg)](http://godoc.org/github.com/madflojo/hord)

Package hord provides a simple and extensible interface for interacting with various database systems in a uniform way.

Hord is designed to be a database-agnostic library that provides a common interface for interacting with different database systems. It allows developers to write code that is decoupled from the underlying database technology, making it easier to switch between databases or support multiple databases in the same application.

## Features

- **Driver-based**: Hord follows a driver-based architecture, where each database system is implemented as a separate driver. This allows for easy extensibility to support new databases.
- **Uniform API**: Hord provides a common API for database operations, including key-value operations, setup, and configuration. The API is designed to be simple and intuitive.
- **Pluggable**: Developers can choose and configure the desired database driver based on their specific needs.
- **Error handling**: Hord provides error types and constants for consistent error handling across drivers.
- **Testing with Mock Driver**: Hord provides a mock driver in the `mock` package, which can be used for testing purposes. The `mock` driver allows users to define custom functions executed when calling the `Database` interface methods, making it easier to test code that relies on the Hord interface.
- **Documentation**: Each driver comes with its own package documentation, providing guidance on how to use and configure the driver.

## Database Drivers:

| Database | Support | Comments | Protocol Compatible Alternatives |
| -------- | ------- | -------- | -------------------------------- |
| [BoltDB](https://github.com/etcd-io/bbolt) | ✅ | | |
| [Cassandra](https://cassandra.apache.org/) | ✅ | | [ScyllaDB](https://www.scylladb.com/), [YugabyteDB](https://www.yugabyte.com/), [Azure Cosmos DB](https://learn.microsoft.com/en-us/azure/cosmos-db/introduction) |
| [Couchbase](https://www.couchbase.com/) | Pending |||
| Hashmap | ✅ | Optionally allows storing to YAML or JSON file ||
| Mock | ✅ | Mock Database interactions within unit tests ||
| [NATS](https://nats.io/) | ✅ | Experimental ||
| [Redis](https://redis.io/) | ✅ || [Dragonfly](https://www.dragonflydb.io/), [KeyDB](https://docs.keydb.dev/) |

## Usage

To use Hord, import it as follows:

    import "github.com/madflojo/hord"

### Creating a Database Client

To create a database client, you need to import and use the appropriate driver package along with the `hord` package.

For example, to use the Redis driver:

```go
import (
    "github.com/madflojo/hord"
    "github.com/madflojo/hord/redis"
)

func main() {
    var db hord.Database
    db, err := redis.Dial(redis.Config{})
    if err != nil {
        // Handle connection error
    }

    // Use the db client for database operations
    // ...
}
```

Each driver provides its own `Dial` function to establish a connection to the database. Refer to the specific driver documentation for more details.

### Database Operations

Once you have a database client, you can use it to perform various database operations. The API is consistent across different drivers.

```go
// Set a value
err = db.Set("key", []byte("value"))
if err != nil {
    // Handle error
}

// Retrieve a value
value, err := db.Get("key")
if err != nil {
    // Handle error
}
```

Refer to the `hord.Database` interface documentation for a complete list of available methods.

## Contributing
Thank you for your interest in helping develop Hord. The time, skills, and perspectives you contribute to this project are valued.

Please reference our [Contributing Guide](CONTRIBUTING.md) for details.

## License
[Apache License 2.0](https://choosealicense.com/licenses/apache-2.0/)
