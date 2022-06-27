package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
)

type index struct {
	Name      string
	IsPrimary bool
	Fields    []string
}

var (
	cluster     *gocb.Cluster
	queryIdxMgr *gocb.QueryIndexManager
	bucketname  *string
)

func main() {
	server := flag.String(
		"server",
		"",
		"The url to the couchbase server",
	)

	bucketname = flag.String(
		"bucket",
		"",
		"The couchbase bucket",
	)

	user := flag.String(
		"user",
		"",
		"The couchbase user",
	)

	pwd := flag.String(
		"pwd",
		"",
		"The couchbase pwd",
	)

	flag.Parse()

	fmt.Printf("server: %s, bucket: %s, user: %s, pwd: %s\n", *server, *bucketname, *user, *pwd)
	var err error
	cluster, err = gocb.Connect(
		*server,
		gocb.ClusterOptions{
			Username: *user,
			Password: *pwd,
		})
	if err != nil {
		panic(err)
	}

	bucket := cluster.Bucket(*bucketname)
	err = bucket.WaitUntilReady(time.Duration(5)*time.Second, nil)
	if err != nil {
		panic(err)
	}

	queryIdxMgr = cluster.QueryIndexes()

	indexes := readFile()

	report := make(map[string]error)
	for _, idx := range indexes {
		err := createIndex(idx)
		if err != nil {
			report[idx.Name] = err
		}
	}

	fmt.Println("Error reporting:")
	for k, v := range report {
		fmt.Printf("%s:\t%s", k, v)
	}
}

func createIndex(idx index) error {
	fmt.Printf("Creating index: %s\n", idx.Name)
	var err error
	if idx.IsPrimary {
		err = queryIdxMgr.CreatePrimaryIndex(*bucketname, &gocb.CreatePrimaryQueryIndexOptions{
			IgnoreIfExists: false,
			CustomName:     idx.Name,
		})
	} else {
		err = queryIdxMgr.CreateIndex(*bucketname, idx.Name, idx.Fields, &gocb.CreateQueryIndexOptions{
			IgnoreIfExists: false,
		})
	}
	return err
}

func readFile() []index {
	dat, err := os.ReadFile(*bucketname + ".json")
	if err != nil {
		panic(err)
	}
	var indexes []index
	err = json.Unmarshal(dat, &indexes)
	if err != nil {
		panic(err)
	}
	return indexes
}
