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

	"log"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/sdslabs/beastv4/core/sidecar/protos/mysql"
)

const MYSQL_AGENT_PORT uint32 = 9500

// Root user credentials for mysql sidecar.
var dbHost = `%`
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
	log.Println("Creating mysql database and user instance")
	connection := fmt.Sprintf("%s:%s@/", dbUser, dbPass)
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

	query := fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY '%s'", username, dbHost, password)
	_, err = db.Exec(query)
	if err != nil {
		return instance, fmt.Errorf("Error while creating user : %s", err)
	}

	query = fmt.Sprintf("GRANT ALL ON %s.* TO '%s'@'%s'", database, username, dbHost)
	_, err = db.Exec(query)
	if err != nil {
		return instance, fmt.Errorf("Error granting permissions to user : %s", err)
	}

	_, err = db.Exec("FLUSH PRIVILEGES")
	if err != nil {
		return instance, fmt.Errorf("Error while flushing user priviliges : %s", err)
	}

	return instance, nil
}

func (s *mysqlAgentServer) DeleteMySQLInstance(ctx context.Context, instance *pb.MySQLInstance) (*pb.None, error) {
	none := &pb.None{}
	connection := fmt.Sprintf("%s:%s@/", dbUser, dbPass)

	db, err := sql.Open("mysql", connection)
	if err != nil {
		return none, fmt.Errorf("Error while connecting to database.")
	}
	defer db.Close()

	_, err = db.Exec("DROP DATABASE " + instance.Database)
	if err != nil {
		return none, fmt.Errorf("Error while deleting the database : %s", err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP USER '%s'@'%s'", instance.Username, dbHost))
	if err != nil {
		return none, fmt.Errorf("Error while deleting the user : %s", err)
	}
	return none, nil
}

func main() {
	listner, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", MYSQL_AGENT_PORT))
	if err != nil {
		fmt.Println("Error while starting listener : %s", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	server := mysqlAgentServer{}
	pb.RegisterMySQLServiceServer(grpcServer, &server)

	fmt.Printf("Starting new server at port : %d", MYSQL_AGENT_PORT)
	grpcServer.Serve(listner)
}
