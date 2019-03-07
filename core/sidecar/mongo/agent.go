package mongo

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"google.golang.org/grpc"

	pb "github.com/sdslabs/beastv4/core/sidecar/protos/mongo"
	log "github.com/sirupsen/logrus"
)

type MongoAgent struct{}

const MONGO_AGENT_PORT uint32 = 9501

var serverAddr string = fmt.Sprintf("127.0.0.1:%d", MONGO_AGENT_PORT)
var opts = []grpc.DialOption{grpc.WithInsecure()}

// This function assumes that store path is the path of the file in which the instance
// values should be store, it assumes that the directory containing the file exist and
// the file represented by configPath does not exists. This function will create the file
// and will write the configuration to the file in json format.
func (a *MongoAgent) Bootstrap(configPath string) error {
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("ERROR while dailing RPC : %s", err)
	}
	defer conn.Close()

	client := pb.NewMongoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	instance, err := client.CreateMongoInstance(ctx, &pb.None{})
	if err != nil {
		return err
	}
	log.Debugf("Created instance in Mongo sidecar with details : %v", instance)
	// Save instance details here in a file or a database, so it can be used later for
	// destroying context through agent.
	instStr, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("Error while marshalling instance for storing: %s", err)
	}

	file, err := os.Create(configPath)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("Error while creating sidecar configuration file: %s", err)
	}

	w := bufio.NewWriter(file)
	_, err = w.WriteString(string(instStr))
	if err != nil {
		return fmt.Errorf("Error while writing sidecar configuration to file: %s", err)
	}
	w.Flush()

	return nil
}

func (a *MongoAgent) Destroy(configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("Error while opening sidecar configuration file: %s", err)
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	var instance pb.MongoInstance
	json.Unmarshal([]byte(byteValue), &instance)

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("ERROR while dailing RPC : %s", err)
	}
	defer conn.Close()

	client := pb.NewMongoServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debugf("Deleting instance in Mongo sidecar with details : %v", instance)
	_, err = client.DeleteMongoInstance(ctx, &instance)
	if err != nil {
		return err
	}

	err = os.Remove(configPath)
	if err != nil {
		return fmt.Errorf("Error while removing undesired sidecar configuration: %s", err)
	}

	return nil
}
