

## Defining object

Let there be a common object definition for all stored entities (Object).
This object will have a unique content identifier (ObjectRef = CID).
This will allow us to store and retrieve objects in a consistent manner across different services.
The Object will have the following structure:

```proto
message Object {
    ObjectRef ref = 1; // Unique content identifier
    bytes data = 2;    // Actual data of the object
}
```
