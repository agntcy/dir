## ObjectManager

This package serves as a decoupled, standalone, non-API interface to handle versioning and conversion between different objects passed across the API with typehints for internal logic.

It allows unified control of objects, regardless of their schemas, versions, and formats; only dependant on their usage across APIs and codebase.
For example, generic objects in the storage layer can be casted to specific types using this interface.

Handlers for new objects can be added in the similar way as done for `Record` types.
All objects use specific `ObjectType` IDs that are used for type embeddings via CID codecs.
An example is provided below.

### Example: Records

This example demonstrates how to use CID codec to embed schema, version, and format of the Record object being managed in a generic way.

```go
package main

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

func main() {
	// Define the generic record object with a specific schema/version/format
	recordType := 1002 // RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON
	recordData := []byte(`{"name": "my-record"}`)

	// Create a cid manually (embeds the object type via codec)
	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(recordType),
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}

	// Create the typed CID from the data
	c, err := pref.Sum(recordData)
	if err != nil {
		panic(err)
	}

    // Print the CID
	fmt.Printf("Record CID(type=%d): %v", recordType, c)
    // Record CID(type=1002): bahvaoerathi53pba7x3sdjchhk4kl5mwgbqex5a2m4yfdbxqvyaixyrnjcha
}
```
