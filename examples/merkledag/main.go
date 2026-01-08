package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
)

const recordFile = "./testdata/record_080.json"

func main() {
	ctx := context.Background()

	// load data from tetdata object
	data, err := os.ReadFile(recordFile)
	if err != nil {
		panic(err)
	}

	// convert to OASF record
	var record *oasfv1.Record
	if err := json.Unmarshal(data, &record); err != nil {
		panic(err)
	}

	// push and pull to local store
	client, err := NewLocalClient()
	if err != nil {
		panic(err)
	}

	// Push data
	desc, err := pushData(ctx, client, record)
	if err != nil {
		panic(err)
	}

	// Pull data
	pulledRecord, err := pullData(ctx, client, desc)
	if err != nil {
		panic(err)
	}

	// ensure pulled record matches original
	pulledData, err := json.MarshalIndent(pulledRecord, "", "  ")
	if err != nil {
		panic(err)
	}

	// save pulled data for inspection
	if err := os.WriteFile("./testdata/pulled_record.json", pulledData, 0644); err != nil {
		panic(err)
	}

	// find diff between the two strings
	if reflect.DeepEqual(record, pulledRecord) == false {
		panic("pulled record does not match original")
	}

}
