package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"text/template"
	"time"

	"google.golang.org/grpc"

	"log"

	pb "github.com/sdslabs/beastv4/core/sidecar/protos/mongo"
)

const MONGO_AGENT_PORT uint32 = 9501

var mongoScript = `
use {{.Database}}
db.createUser(
        {
                user:   "{{.Username}}",
                pwd:    "{{.Password}}",
                roles:
                [
                        {
                                role:   "readWrite",
                                db:     "{{.Database}}"
                        }
                ]
        }
)
db.createCollection("test");
`

// Root user credentials for mongo sidecar.
var dbUser = os.Getenv("MONGO_INITDB_ROOT_USERNAME")
var dbPass = os.Getenv("MONGO_INITDB_ROOT_PASSWORD")
var createCommand = fmt.Sprintf("mongo admin -u %s -p %s < /tmp/createUser.js", dbUser, dbPass)

type mongoAgentServer struct{}

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

func (s *mongoAgentServer) CreateMongoInstance(ctx context.Context, none *pb.None) (*pb.MongoInstance, error) {
	log.Println("Creating mongo database and user instance")
	database := randString(16)
	username := randString(16)
	password := randString(16)

	instance := &pb.MongoInstance{
		Username: username,
		Database: database,
		Password: password,
	}
	file, err := os.OpenFile("/tmp/createUser.js", os.O_CREATE|os.O_WRONLY, 0644)

	var buff bytes.Buffer
	mongoTemplate, err := template.New("mongoscript").Parse(mongoScript)
	if err != nil {
		file.Close()
		return instance, fmt.Errorf("Error while parsing template :: %s", err)
	}

	err = mongoTemplate.Execute(&buff, instance)
	if err != nil {
		file.Close()
		return instance, fmt.Errorf("Error while executing template :: %s", err)
	}

	_, err = file.WriteString(buff.String())
	if err != nil {
		file.Close()
		return instance, fmt.Errorf("Error while writing to file :: %s", err)
	}

	file.Close()
	err = exec.Command("bash", "-c", createCommand).Run()
	if err != nil {
		return instance, fmt.Errorf("Error while running command: %v", err)
	}

	return instance, nil
}

func (s *mongoAgentServer) DeleteMongoInstance(ctx context.Context, instance *pb.MongoInstance) (*pb.None, error) {
	none := &pb.None{}

	err := exec.Command("sh", "-c", fmt.Sprintf("mongo admin -u %s -p %s --eval \"db=db.getSiblingDB('%s');db.dropDatabase();\"", dbUser, dbPass, instance.Database)).Run()
	if err != nil {
		return none, fmt.Errorf("Error while deleting the database : %s", err)
	}

	return none, nil
}

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", MONGO_AGENT_PORT))
	if err != nil {
		fmt.Println("Error while starting listner : %s", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	server := mongoAgentServer{}
	pb.RegisterMongoServiceServer(grpcServer, &server)

	fmt.Printf("Starting new server at port : %d", MONGO_AGENT_PORT)
	grpcServer.Serve(listener)
}
