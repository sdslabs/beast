package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	pb "github.com/sdslabs/beastv4/extras/agents/mysql/protobuf"
)

const MYSQL_AGENT_PORT uint32 = 9500

// Root user credentials for mysql sidecar.
var dbHost = "localhost"
var dbUser = "root"
var dbPass = os.Getenv("MYSQL_ROOT_PASSWORD")

type mysqlAgentServer struct{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (s *mysqlAgentServer) CreateMySQLInstance(ctx context.Context, none *pb.None) (*pb.MySQLInstance, error) {
	connection := fmt.Sprintf("%s:%s@tcp(%s:3306)/", dbUser, dbPass, dbHost)
	database := randString(16)
	username := randString(16)
	password := randString(16)

	instance := &pb.MySQLInstance{
		Username: username,
		Database: database,
		Password: password,
	}

	db, err := sql.Open("mysql", connection)
	if err != nil {
		return instance, fmt.Errorf("Error while connecting to database.")
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE " + database)
	if err != nil {
		return instance, fmt.Errorf("Error while creating the database : %s", err)
	}

	_, err = db.Exec("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%s' IDENTIFIED BY '%s'", database, username, dbHost, password)
	if err != nil {
		return instance, fmt.Errorf("Error while creating user : %s", err)
	}

	_, err = db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		return instance, fmt.Errorf("Error while flushing user priviliges : %s", err)
	}

	return instance, nil
}

func (s *mysqlAgentServer) DeleteMySQLInstance(ctx context.Context, instance *pb.MySQLInstance) (*pb.None, error) {
	none := &pb.None{}
	connection := fmt.Sprintf("%s:%s@tcp(%s:3306)/", dbUser, dbPass, dbHost)

	db, err := sql.Open("mysql", connection)
	if err != nil {
		return none, fmt.Errorf("Error while connecting to database.")
	}
	defer db.Close()

	_, err = db.Exec("DELETE DATABASE " + instance.Database)
	if err != nil {
		return none, fmt.Errorf("Error while deleting the database : %s", err)
	}

	_, err = db.Exec("DROP USER '%s'@'%s'", instance.Username, dbHost)
	if err != nil {
		return none, fmt.Errorf("Error while deleting the user : %s", err)
	}
	return none, nil
}

func main() {
	listner, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", MYSQL_AGENT_PORT))
	if err != nil {
		fmt.Println("Error while starting listner : %s", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	server := mysqlAgentServer{}
	pb.RegisterMySQLServiceServer(grpcServer, &server)

	fmt.Printf("Starting new server at port : %d", MYSQL_AGENT_PORT)
	grpcServer.Serve(listner)
}
