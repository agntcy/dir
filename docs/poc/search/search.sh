#!/bin/bash

query='
{
  ExpandedRepoInfo(repo: \"demo\") {
    Images {
      Tag
      Digest
    }
  }
}
'
query="$(echo $query)"   # the query should be a one-liner, without newlines

curl -H 'Content-Type: application/json' \
   -X POST -d "{ \"query\": \"$query\"}" http://localhost:5000/v2/_zot/ext/search | jq .
