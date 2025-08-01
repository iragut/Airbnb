# Airbnb copy

## About
Implement a simple Airbnb clone using Go for backend and html/css for frontend. For database, is using MySQL docker container.
It contains basic features like user registration, listing creation, profile editing, booking, search, etc.
Use Cookie for session management and GET/POST methods for API calls. It connect to MySQL database to store user and listing data.
If it first time running, it will create the necessary tables in the database and some profile listenings for testing.

![frontend](/static/images/front.png)

## Running the project
1. Clone the repository:
2. Install Go and MySQL, docker.
3. Create a MySQL docker container:
    ```bash
    docker run --name mysql-airbnb -e MYSQL_ROOT_PASSWORD=example -e MYSQL_DATABASE=mysql-airbnb  -p 3306:3306 -d mysql
    ```
4. Then run the project in the terminal:
    ```bash
        go run .
    ```
5. Open your browser and go to `http://localhost:8080`.


## Requirements:
- Go 1.18 or later
- MySQL 5.7 or later
- Docker (for MySQL container)
