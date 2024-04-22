# Tezos delegation watcher

This project is designed to monitor delegations on the Tezos blockchain in real-time. It implements a producer-consumer model where:

- **Producer**: Subscribes to blockchain events and fetches new delegations data.
- **Consumer**: Processes the fetched data and saves it to a PostgreSQL database.


## Requirements

- go 1.21.3
- Postgresql 
- Docker (optional, for running PostgreSQL locally)

## Getting Started

### Configuration

Before running the application, you need to set up your `config.yaml` with appropriate values, such as database credentials and API endpoints.

### Running PostgreSQL using Docker (Optional)

If you do not have a PostgreSQL server, you can start one using Docker:

```bash
docker-compose up -d
```


### Architecture
The Tezos Delegation Watcher uses a producer-consumer architecture to enhance data processing efficiency and maintainability:

**Producer**: Monitors the blockchain for new heads using the Tzkt API's WebSocket service. When a new head is detected, the producer fetches all relevant delegation operations from that block and sends them to a processing channel.

**Consumer** : Listens on the processing channel for new data to process. This includes storing new delegations in the database and handling blockchain reorganizations when necessary.


This separation of concerns ensures that the system is robust against individual component failures and can be scaled to handle high volumes of blockchain events.




### Build the Application

Compile the application to ensure everything is set up correctly:

```bash
make build
```

### Run Tests

Execute the tests to verify that all components function correctly:

```bash
make test
```

### Running the Application

Start the watcher using:

```bash
make run
```





