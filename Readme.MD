# Viswals Backend Test

This project demonstrates a robust backend system that processes data from a CSV file by publishing it to a RabbitMQ queue. A consumer reads the data from the queue, processes it, and stores it in Redis (for caching) and PostgreSQL (for persistence). Additionally, a REST API is provided to retrieve and manage the stored data, ensuring efficient and scalable data handling.

---

## Project Overview

### Producer

The producer is responsible for reading data from a CSV file, parsing it into a structured JSON format, and publishing it to RabbitMQ for further processing.

#### Key Tasks:
1. **Read Data**: Reads data from a CSV file.
2. **Parse Data**: Converts the CSV data into structured JSON.
3. **Publish Messages**: Connects to RabbitMQ, declares a queue, and publishes the parsed data.

---

### Consumer

The consumer listens to RabbitMQ for incoming messages, processes the data, and stores it in Redis and PostgreSQL. It also exposes RESTful APIs for interacting with the stored data.

#### Key Tasks:
1. **Consume Messages**: Listens to RabbitMQ for new messages.
2. **Process Data**: Parses the data into a structured format.
3. **Redis Caching**: Stores processed data in Redis for quick access.
4. **PostgreSQL Storage**: Persists the data in PostgreSQL for long-term storage.
5. **API Service**: Exposes REST APIs for retrieving and managing stored data.

---

## API Documentation

### Available Endpoints:
| **API Name**          | **HTTP Method** | **Endpoint**        | **Description**                                                     |
|------------------------|-----------------|---------------------|----------------------------------------------------------------------|
| **Get All Users**      | `GET`          | `/users`            | Fetch a list of all users stored in the database.                    |
| **Get User by ID**     | `GET`          | `/users/{id}`       | Retrieve detailed information for a specific user by their ID.       |
| **Get All Users (SSE)**| `GET`          | `/users/sse`        | Stream all users to the client in real-time using Server-Sent Events.|
| **Create User**        | `POST`         | `/users`            | Add a new user to the database.                                      |
| **Delete User**        | `DELETE`       | `/users/{id}`       | Remove a user from the database by their ID.                         |

---

## How to Run

Follow these steps to build, containerize, and run the application in a Dockerized environment:

1. **Start the Application**:
   Run the following command in the project root directory:
   ```bash
   docker-compose up --build

2. **Access the API**:
   Once the containers are up and running, you can access the API at `http://localhost:8080`.

3. **Verify Services**:
   Use tools like curl or Postman to test API endpoints. For RabbitMQ, you can verify the queue status by accessing the RabbitMQ management console at http://localhost:15672.

## Additional Notes
   Ensure that all environment variables in the docker-compose.yml file are properly configured before starting the application.



