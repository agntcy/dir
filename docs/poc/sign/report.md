Artifact tree: 
```
Ref: localhost:5000/demo:artifact1
Digest: sha256:2f0249ebb962c1281b791aafe68ec39caa2751eecad1c6a5042cbdc43548d764
Referrers:
  - sha256:88afe1b392119ad3d861c84fbd93b86a0404d209fd27c5adf11deca8c11410c7: application/vnd.dev.cosign.artifact.sig.v1+json
```

Signature digest: 
```
sha256:88afe1b392119ad3d861c84fbd93b86a0404d209fd27c5adf11deca8c11410c7
```

Signature artifact: 
```json
{
    "critical": {
        "identity": {
            "docker-reference": "localhost:5000/demo"
        },
        "image": {
            "docker-manifest-digest": "sha256:2f0249ebb962c1281b791aafe68ec39caa2751eecad1c6a5042cbdc43548d764"
        },
        "type": "cosign container image signature"
    },
    "optional": null
}
```

Signature blob: 
```json
{
    "schemaVersion": 2,
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "config": {
        "mediaType": "application/vnd.dev.cosign.artifact.sig.v1+json",
        "size": 233,
        "digest": "sha256:412c691461c6af36c6c5f2412926003d8386bc9e3dfbf54823652d59d4617458"
    },
    "layers": [
        {
            "mediaType": "application/vnd.dev.cosign.simplesigning.v1+json",
            "size": 235,
            "digest": "sha256:bc59dffae964a8135399a8bc1897d79ac216b01007f52cb8e96a053d04395613",
            "annotations": {
                "dev.cosignproject.cosign/signature": "MEYCIQDXkREZ7yk7UEYMoYOhgl8LACNeEoWI3zIkc7EM54TolQIhANYkfFQR7OTWRI0kE3w/T1niR8rfKRc4MVFqOeUgoET7",
                "dev.sigstore.cosign/bundle": "{\"SignedEntryTimestamp\":\"MEYCIQDbEDDWL1V1g8wFf0Z5FXiz/CW9v21uaINxQdBGOVRGcgIhAI/MzDV/qUx+d96qnhpx6smUQUfvvEhQRUfYlsIwQ3RR\",\"Payload\":{\"body\":\"eyJhcGlWZXJzaW9uIjoiMC4wLjEiLCJraW5kIjoiaGFzaGVkcmVrb3JkIiwic3BlYyI6eyJkYXRhIjp7Imhhc2giOnsiYWxnb3JpdGhtIjoic2hhMjU2IiwidmFsdWUiOiJiYzU5ZGZmYWU5NjRhODEzNTM5OWE4YmMxODk3ZDc5YWMyMTZiMDEwMDdmNTJjYjhlOTZhMDUzZDA0Mzk1NjEzIn19LCJzaWduYXR1cmUiOnsiY29udGVudCI6Ik1FWUNJUURYa1JFWjd5azdVRVlNb1lPaGdsOExBQ05lRW9XSTN6SWtjN0VNNTRUb2xRSWhBTllrZkZRUjdPVFdSSTBrRTN3L1QxbmlSOHJmS1JjNE1WRnFPZVVnb0VUNyIsInB1YmxpY0tleSI6eyJjb250ZW50IjoiTFMwdExTMUNSVWRKVGlCUVZVSk1TVU1nUzBWWkxTMHRMUzBLVFVacmQwVjNXVWhMYjFwSmVtb3dRMEZSV1VsTGIxcEplbW93UkVGUlkwUlJaMEZGYUdNM2MwSlNUMlY2SzBoWlNHUTRiMUJIUW5sWVlWTTJRVFo0ZEFwVlRHcE9lamhWTlc1SFkwTkxVa05RVHk5RGJFeFhWMHB2U0hwblNtbElkVVpyYld4b2VVRk1RMWREWjJvM09GSXdibXRzV0RsTVpUSm5QVDBLTFMwdExTMUZUa1FnVUZWQ1RFbERJRXRGV1MwdExTMHRDZz09In19fX0=\",\"integratedTime\":1754488662,\"logIndex\":357097618,\"logID\":\"c0d23d6ad406973f9559f3ba2d1ca01f84147d8ffc5b8445c224f98b9591801d\"}}"
            }
        }
    ],
    "subject": {
        "mediaType": "application/vnd.oci.image.manifest.v1+json",
        "size": 652,
        "digest": "sha256:2f0249ebb962c1281b791aafe68ec39caa2751eecad1c6a5042cbdc43548d764"
    }
}
```


Zot GraphQL Response body:
```json
{
    "data": {
        "Image": {
            "Digest": "sha256:e551cb42e49f378747fad1506376b94fce5718cd57346fe5f400a0e198292637",
            "IsSigned": true,
            "Tag": "artifact1",
            "SignatureInfo": [
                {
                    "Tool": "cosign",
                    "IsTrusted": true,
                    "Author": "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEcIGJxQWgKCRYsLpyYQOHbAUavMh4\nvwarTP0ysYRcOBGy1FIwYYyvRJDoByR/OE0h0I2dIEg71bJQK/Q/lPvBCA==\n-----END PUBLIC KEY-----\n"
                }
            ]
        }
    }
}
```