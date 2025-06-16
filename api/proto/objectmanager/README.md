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
	// Step 1: Define the generic object
	// CID codec embeds a specific object schema, version, and format
	// Data that needs to be e.g. stored, validated, or parsed for a specific object
	objectType := 1002                            // RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON
	objectData := []byte(`{"name": "my-record"}`) // Record object

	// Step 2: Create a CID for the object (embed the object type via codec)
	cidPref := cid.Prefix{
		Version:  1,
		Codec:    uint64(objectType),
		MhType:   mh.SHA2_256,
		MhLength: -1,
	}

	objectCID, err := cidPref.Sum(objectData)
	if err != nil {
		panic(err)
	}

	// Step 3: Print the object CID
	fmt.Printf("Object CID(type=%d): %v", objectType, objectCID)

	// Step 4: Extract the object type from CID
	// Step 5: Parse the data depending on the object type
	// Step 6: Concrete object available
}
```
