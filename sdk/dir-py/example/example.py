# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from google.protobuf.json_format import MessageToJson

from agntcy_dir.client import Client, Config
from agntcy_dir.models import *


def generate_record(name):
    return core_v1.Record(
        v3=objects_v3.Record(
            name=name,
            schema_version="v0.5.0",
            skills=[
                objects_v3.Skill(
                    name="Natural Language Processing",
                ),
            ],
            locators=[
                objects_v3.Locator(
                    type="ipv4",
                    url="127.0.0.1",
                ),
            ],
            extensions=[
                objects_v3.Extension(
                    name="runtime/prompt",
                    version="v1",
                ),
            ],
            signature=objects_v3.Signature(),
        ),
    )


def main() -> None:
    # Initialize the client
    client = Client(Config())

    records = [generate_record(x) for x in ["example-record", "example-record2"]]

    # Push objects to the store
    refs = client.push(records)

    for ref in refs:
        print("Pushed object ref:", ref.cid)

    # Pull objects from the store
    pulled_records = client.pull(refs)

    for pulled_record in pulled_records:
        print("Pulled object data:", MessageToJson(pulled_record))

    # Lookup the object
    metadatas = client.lookup(refs)

    for metadata in metadatas:
        print("Lookup object metadata:", MessageToJson(metadata))

    # Publish the object
    record_refs = routing_v1.RecordRefs(refs=[refs[0]])
    publish_request = routing_v1.PublishRequest(record_refs=record_refs)
    client.publish(publish_request)
    print("Object published.")

    # List objects in the store
    query = routing_v1.RecordQuery(
        type=routing_v1.RECORD_QUERY_TYPE_SKILL,
        value="/skills/Natural Language Processing/Text Completion",
    )

    list_request = routing_v1.ListRequest(queries=[query])
    objects = list(client.list(list_request))

    for o in objects:
        print("Listed object:", MessageToJson(o))

    # Search objects
    search_query = search_v1.RecordQuery(
        type=search_v1.RECORD_QUERY_TYPE_SKILL_ID, value="1",
    )

    search_request = search_v1.SearchRequest(queries=[search_query], limit=3)
    objects = list(client.search(search_request))

    print("Searched objects:",objects)

    # Unpublish the object
    record_refs = routing_v1.RecordRefs(refs=[refs[0]])
    unpublish_request = routing_v1.UnpublishRequest(record_refs=record_refs)
    client.unpublish(unpublish_request)
    print("Object unpublished.")

    # Delete the object
    client.delete(refs)
    print("Objects are deleted.")


if __name__ == "__main__":
    main()
